package scm

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

var lookupIPAddr = net.DefaultResolver.LookupIPAddr

func ValidateGitRemoteURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty git url")
	}
	host, err := gitRemoteHost(raw)
	if err != nil {
		return err
	}
	return validateRemoteHost(host)
}

func gitRemoteHost(raw string) (string, error) {
	if strings.Contains(strings.ToLower(raw), "file://") {
		return "", fmt.Errorf("file:// remotes not allowed")
	}
	if strings.HasPrefix(strings.ToLower(raw), "git@") {
		host, path, ok := splitGitSSHRemote(raw)
		if !ok {
			return "", fmt.Errorf("invalid git ssh url")
		}
		if strings.Contains(path, "..") {
			return "", fmt.Errorf("invalid characters in git ssh url")
		}
		return host, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		host := strings.TrimSpace(u.Hostname())
		if host == "" {
			return "", fmt.Errorf("git remote host required")
		}
		return host, nil
	default:
		return "", fmt.Errorf("unsupported scheme %q (use http(s) or git@host:…)", u.Scheme)
	}
}

func splitGitSSHRemote(raw string) (host string, path string, ok bool) {
	rest := raw[len("git@"):]
	if strings.HasPrefix(rest, "[") {
		end := strings.Index(rest, "]")
		if end <= 1 || end+1 >= len(rest) || rest[end+1] != ':' {
			return "", "", false
		}
		host = rest[1:end]
		path = rest[end+2:]
	} else {
		colon := strings.Index(rest, ":")
		if colon <= 0 || colon+1 >= len(rest) {
			return "", "", false
		}
		host = rest[:colon]
		path = rest[colon+1:]
	}
	host = strings.TrimSpace(host)
	path = strings.TrimSpace(path)
	if host == "" || path == "" || strings.ContainsAny(host, "/\\ \t\r\n") {
		return "", "", false
	}
	return host, path, true
}

func validateRemoteHost(rawHost string) error {
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(rawHost)), ".")
	if host == "" {
		return fmt.Errorf("git remote host required")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "metadata.google.internal" {
		return fmt.Errorf("disallowed host %q", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isDisallowedRemoteIP(ip) {
			return fmt.Errorf("disallowed host %q", host)
		}
		return nil
	}
	if looksLikeIPv4Literal(host) {
		return fmt.Errorf("invalid numeric git host %q", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	addrs, err := lookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve git host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("resolve git host %q: no addresses", host)
	}
	for _, addr := range addrs {
		if isDisallowedRemoteIP(addr.IP) {
			return fmt.Errorf("disallowed host %q resolves to %s", host, addr.IP.String())
		}
	}
	return nil
}

func isDisallowedRemoteIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() ||
		!ip.IsGlobalUnicast()
}

func looksLikeIPv4Literal(host string) bool {
	if host == "" {
		return false
	}
	for _, r := range host {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
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

	cmd := exec.CommandContext(ctx, "git", "-c", "http.followRedirects=false", "-c", "protocol.file.allow=never", "ls-remote", "--heads", repoURL)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
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
