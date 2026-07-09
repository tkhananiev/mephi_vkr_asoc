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
	"strconv"
	"strings"
	"time"
)

func ValidateGitRemoteURL(raw string) error {
	return validateGitRemoteURL(raw, net.DefaultResolver.LookupIPAddr)
}

type ipResolver func(context.Context, string) ([]net.IPAddr, error)

var disallowedHostPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
}

func validateGitRemoteURL(raw string, resolve ipResolver) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty git url")
	}
	if strings.Contains(strings.ToLower(raw), "file://") {
		return fmt.Errorf("file:// remotes not allowed")
	}
	if strings.HasPrefix(strings.ToLower(raw), "git@") {
		host, err := gitSSHHost(raw)
		if err != nil {
			return err
		}
		if err := validateRemoteHost(host, resolve); err != nil {
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
		lowerHost = u.Hostname()
	default:
		return fmt.Errorf("unsupported scheme %q (use http(s) or git@host:…)", u.Scheme)
	}
	return validateRemoteHost(lowerHost, resolve)
}

func gitSSHHost(raw string) (string, error) {
	rest := strings.TrimSpace(raw[len("git@"):])
	host, _, ok := strings.Cut(rest, ":")
	if !ok || strings.TrimSpace(host) == "" {
		return "", fmt.Errorf("invalid git ssh url")
	}
	return strings.Trim(host, "[]"), nil
}

func validateRemoteHost(host string, resolve ipResolver) error {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return fmt.Errorf("empty git host")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "metadata.google.internal" {
		return fmt.Errorf("disallowed host %q", host)
	}
	if addr, ok := parseRemoteHostIP(host); ok {
		return validateRemoteIP(host, addr)
	}
	if resolve == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := resolve(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("resolve host %q: no addresses", host)
	}
	for _, ipAddr := range addrs {
		addr, ok := netip.AddrFromSlice(ipAddr.IP)
		if !ok {
			return fmt.Errorf("resolve host %q: invalid address %q", host, ipAddr.IP.String())
		}
		if err := validateRemoteIP(host, addr.Unmap()); err != nil {
			return err
		}
	}
	return nil
}

func parseRemoteHostIP(host string) (netip.Addr, bool) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Unmap(), true
	}
	return parseLegacyIPv4(host)
}

func parseLegacyIPv4(host string) (netip.Addr, bool) {
	parts := strings.Split(host, ".")
	if len(parts) == 0 || len(parts) > 4 {
		return netip.Addr{}, false
	}
	nums := make([]uint64, len(parts))
	for i, part := range parts {
		if part == "" {
			return netip.Addr{}, false
		}
		n, err := strconv.ParseUint(part, 0, 32)
		if err != nil {
			return netip.Addr{}, false
		}
		nums[i] = n
	}

	var value uint64
	switch len(nums) {
	case 1:
		if nums[0] > 0xffffffff {
			return netip.Addr{}, false
		}
		value = nums[0]
	case 2:
		if nums[0] > 0xff || nums[1] > 0xffffff {
			return netip.Addr{}, false
		}
		value = nums[0]<<24 | nums[1]
	case 3:
		if nums[0] > 0xff || nums[1] > 0xff || nums[2] > 0xffff {
			return netip.Addr{}, false
		}
		value = nums[0]<<24 | nums[1]<<16 | nums[2]
	case 4:
		for _, n := range nums {
			if n > 0xff {
				return netip.Addr{}, false
			}
		}
		value = nums[0]<<24 | nums[1]<<16 | nums[2]<<8 | nums[3]
	default:
		return netip.Addr{}, false
	}

	return netip.AddrFrom4([4]byte{
		byte(value >> 24),
		byte(value >> 16),
		byte(value >> 8),
		byte(value),
	}), true
}

func validateRemoteIP(host string, addr netip.Addr) error {
	if !addr.IsGlobalUnicast() || addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsMulticast() || addr.IsUnspecified() {
		return fmt.Errorf("disallowed host %q", host)
	}
	for _, prefix := range disallowedHostPrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("disallowed host %q", host)
		}
	}
	return nil
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
