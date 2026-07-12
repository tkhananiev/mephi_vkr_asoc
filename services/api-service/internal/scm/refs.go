package scm

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const maxLsRemoteOutputBytes = 4 * 1024 * 1024

type hostLookupFunc func(context.Context, string) ([]net.IPAddr, error)

var disallowedRemoteIPPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func ValidateGitRemoteURL(raw string) error {
	return validateGitRemoteURL(context.Background(), raw, net.DefaultResolver.LookupIPAddr)
}

func validateGitRemoteURL(ctx context.Context, raw string, lookup hostLookupFunc) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty git url")
	}
	if strings.Contains(strings.ToLower(raw), "file://") {
		return fmt.Errorf("file:// remotes not allowed")
	}
	host, err := remoteHost(raw)
	if err != nil {
		return err
	}
	return validateRemoteHost(ctx, host, lookup)
}

func remoteHost(raw string) (string, error) {
	if strings.HasPrefix(strings.ToLower(raw), "git@") {
		rest := raw[len("git@"):]
		colon := strings.IndexByte(rest, ':')
		if colon <= 0 || colon == len(rest)-1 {
			return "", fmt.Errorf("invalid git ssh url")
		}
		if strings.Contains(raw, "..") {
			return "", fmt.Errorf("invalid characters in git ssh url")
		}
		return rest[:colon], nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		host := u.Hostname()
		if host == "" {
			return "", fmt.Errorf("missing git remote host")
		}
		return host, nil
	default:
		return "", fmt.Errorf("unsupported scheme %q (use http(s) or git@host:…)", u.Scheme)
	}
}

func validateRemoteHost(ctx context.Context, host string, lookup hostLookupFunc) error {
	host = normalizeRemoteHost(host)
	if host == "" {
		return fmt.Errorf("missing git remote host")
	}
	if strings.ContainsAny(host, "/\\") {
		return fmt.Errorf("invalid git remote host %q", host)
	}
	if disallowedRemoteHostname(host) {
		return fmt.Errorf("disallowed host %q", host)
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		addr = addr.Unmap()
		if disallowedRemoteIP(addr) {
			return fmt.Errorf("disallowed host %q", host)
		}
		return nil
	}
	ips, err := lookup(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve git host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("resolve git host %q: no addresses", host)
	}
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok {
			return fmt.Errorf("resolve git host %q: invalid address %q", host, ip.IP.String())
		}
		addr = addr.Unmap()
		if disallowedRemoteIP(addr) {
			return fmt.Errorf("disallowed host %q resolves to %s", host, addr.String())
		}
	}
	return nil
}

func normalizeRemoteHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return strings.TrimSuffix(host, ".")
}

func disallowedRemoteHostname(host string) bool {
	return host == "localhost" || host == "metadata.google.internal"
}

func disallowedRemoteIP(addr netip.Addr) bool {
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified() {
		return true
	}
	for _, prefix := range disallowedRemoteIPPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func ListRemoteBranches(ctx context.Context, repoURL string) ([]string, error) {
	repoURL = strings.TrimSpace(repoURL)
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
	}
	if err := validateGitRemoteURL(ctx, repoURL, net.DefaultResolver.LookupIPAddr); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "git", "-c", "http.followRedirects=false", "ls-remote", "--heads", repoURL)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := limitedCommandOutput(cmd, maxLsRemoteOutputBytes)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("git ls-remote timed out")
		}
		return nil, err
	}
	return parseLsRemoteHeads(out), nil
}

func limitedCommandOutput(cmd *exec.Cmd, maxBytes int64) ([]byte, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote failed: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git ls-remote failed: %w", err)
	}
	out, tooLarge, readErr := readLimited(stdout, maxBytes)
	if tooLarge {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("git ls-remote output exceeded %d bytes", maxBytes)
	}
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, fmt.Errorf("git ls-remote failed: %w", readErr)
	}
	if waitErr != nil {
		return nil, fmt.Errorf("git ls-remote failed: %w", waitErr)
	}
	return out, nil
}

func readLimited(r io.Reader, maxBytes int64) ([]byte, bool, error) {
	out, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(out)) > maxBytes {
		return nil, true, nil
	}
	return out, false, nil
}

func parseLsRemoteHeads(out []byte) []string {
	seen := make(map[string]struct{})
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		const prefix = "refs/heads/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(ref, prefix))
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	outNames := make([]string, 0, len(seen))
	for name := range seen {
		outNames = append(outNames, name)
	}
	sort.Strings(outNames)
	return outNames
}
