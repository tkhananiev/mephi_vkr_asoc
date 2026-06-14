package scm

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type lookupNetIPFunc func(context.Context, string, string) ([]netip.Addr, error)

func ValidateGitRemoteURL(raw string) error {
	return validateGitRemoteURL(raw, net.DefaultResolver.LookupNetIP)
}

func validateGitRemoteURL(raw string, lookup lookupNetIPFunc) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty git url")
	}
	if strings.Contains(strings.ToLower(raw), "file://") {
		return fmt.Errorf("file:// remotes not allowed")
	}
	if strings.HasPrefix(strings.ToLower(raw), "git@") {
		host, ok := parseSCPStyleGitHost(raw)
		if !ok {
			return fmt.Errorf("invalid git ssh url")
		}
		if err := validateRemoteHost(host, lookup); err != nil {
			return err
		}
		if strings.Contains(raw, "..") {
			return fmt.Errorf("invalid characters in git ssh url")
		}
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	var lowerHost string
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		lowerHost = strings.ToLower(u.Hostname())
	default:
		return fmt.Errorf("unsupported scheme %q (use http(s) or git@host:…)", u.Scheme)
	}
	if err := validateRemoteHost(lowerHost, lookup); err != nil {
		return err
	}
	return nil
}

func parseSCPStyleGitHost(raw string) (string, bool) {
	rest := raw[len("git@"):]
	host, _, ok := strings.Cut(rest, ":")
	if !ok || strings.TrimSpace(host) == "" {
		return "", false
	}
	return host, true
}

func validateRemoteHost(host string, lookup lookupNetIPFunc) error {
	lowerHost := strings.ToLower(strings.TrimSpace(host))
	if lowerHost == "" {
		return fmt.Errorf("empty git host")
	}
	if lowerHost == "localhost" || lowerHost == "metadata.google.internal" {
		return fmt.Errorf("disallowed host %q", lowerHost)
	}

	if addr, err := netip.ParseAddr(lowerHost); err == nil {
		if isDisallowedRemoteIP(addr) {
			return fmt.Errorf("disallowed host %q", lowerHost)
		}
		return nil
	}

	addrs, err := lookup(context.Background(), "ip", lowerHost)
	if err != nil {
		return fmt.Errorf("resolve git host %q: %w", lowerHost, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("resolve git host %q: no addresses", lowerHost)
	}
	for _, addr := range addrs {
		if isDisallowedRemoteIP(addr) {
			return fmt.Errorf("disallowed host %q", lowerHost)
		}
	}
	return nil
}

func isDisallowedRemoteIP(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsUnspecified() ||
		addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() ||
		addr.IsMulticast()
}

func ListRemoteBranches(ctx context.Context, repoURL string) ([]string, error) {
	repoURL = strings.TrimSpace(repoURL)
	if err := ValidateGitRemoteURL(repoURL); err != nil {
		return nil, err
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", repoURL)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.followRedirects",
		"GIT_CONFIG_VALUE_0=false",
	)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("git ls-remote timed out")
		}
		return nil, fmt.Errorf("git ls-remote failed: %w", err)
	}
	return parseLsRemoteHeads(out), nil
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
