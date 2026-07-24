package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ValidateGitRemoteURL(raw string) error {
	return validateGitRemoteURL(context.Background(), raw, net.DefaultResolver.LookupIPAddr)
}

func validateGitRemoteURL(ctx context.Context, raw string, lookup func(context.Context, string) ([]net.IPAddr, error)) error {
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

func validateRemoteHost(ctx context.Context, host string, lookup func(context.Context, string) ([]net.IPAddr, error)) error {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return fmt.Errorf("missing git remote host")
	}
	if host == "localhost" || host == "metadata.google.internal" {
		return fmt.Errorf("disallowed host %q", host)
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if disallowedIP(addr.Unmap()) {
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
			return fmt.Errorf("resolve git host %q: invalid address", host)
		}
		if disallowedIP(addr.Unmap()) {
			return fmt.Errorf("disallowed host %q resolves to %s", host, addr.String())
		}
	}
	return nil
}

func disallowedIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return true
	}
	return !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified()
}

func SecureSubdir(repoRoot string, relative string) (string, error) {
	rr := filepath.Clean(repoRoot)
	rel := strings.Trim(strings.TrimSpace(strings.ReplaceAll(relative, "\\", "/")), "/")
	if rel == "" || rel == "." {
		return assertInsideRoot(rr, rr)
	}
	part := filepath.Clean(filepath.FromSlash(rel))
	if part == "." {
		return assertInsideRoot(rr, rr)
	}
	if part == ".." || strings.HasPrefix(part, ".."+string(filepath.Separator)) || filepath.IsAbs(part) {
		return "", fmt.Errorf("invalid subdirectory %q", relative)
	}
	out := filepath.Clean(filepath.Join(rr, part))
	rrSep := rr + string(filepath.Separator)
	if out != rr && !strings.HasPrefix(out, rrSep) {
		return "", fmt.Errorf("subdirectory escapes clone root")
	}
	return assertInsideRoot(rr, out)
}

func assertInsideRoot(root, path string) (string, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolvedRoot = filepath.Clean(root)
	}
	rootSep := resolvedRoot + string(filepath.Separator)
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		cleaned := filepath.Clean(path)
		if cleaned != resolvedRoot && !strings.HasPrefix(cleaned, rootSep) {
			return "", fmt.Errorf("subdirectory escapes clone root")
		}
		return cleaned, nil
	}
	if resolvedPath != resolvedRoot && !strings.HasPrefix(resolvedPath, rootSep) {
		return "", fmt.Errorf("subdirectory escapes clone root")
	}
	return resolvedPath, nil
}

// AssertPathUnderRoot ensures path (after symlink resolution) stays under allowedRoot.
func AssertPathUnderRoot(path, allowedRoot string) (string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	allowedRoot = filepath.Clean(strings.TrimSpace(allowedRoot))
	if path == "" || path == "." {
		return "", fmt.Errorf("empty target path")
	}
	if allowedRoot == "" || allowedRoot == "." {
		return "", fmt.Errorf("allowed scan root is not configured")
	}
	return assertInsideRoot(allowedRoot, path)
}

func randomSuffix() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

func PrepareGitWorkspace(ctx context.Context, workRoot string, repoURL string, gitRef string, subDirInRepo string) (scanDir string, cleanup func(), err error) {
	if err := ValidateGitRemoteURL(repoURL); err != nil {
		return "", nil, err
	}
	workRoot = filepath.Clean(strings.TrimSpace(workRoot))
	if workRoot == "" {
		workRoot = os.TempDir()
	}
	if err := os.MkdirAll(workRoot, 0750); err != nil {
		return "", nil, err
	}

	dest := filepath.Join(workRoot, "clone-"+randomSuffix())

	args := []string{"clone", "--depth", "1", "--single-branch"}
	ref := strings.TrimSpace(gitRef)
	if ref != "" {
		if strings.HasPrefix(ref, "-") {
			_ = os.RemoveAll(dest)
			return "", nil, fmt.Errorf("invalid git ref %q", gitRef)
		}
		args = append(args, "--branch", ref)
	}
	args = append(args, repoURL, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, cerr := cmd.CombinedOutput()

	if cerr != nil && ref != "" {
		_ = os.RemoveAll(dest)
		cmd2 := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--single-branch", repoURL, dest)
		cmd2.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		out2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			return "", nil, fmt.Errorf("git clone: %w (%s)", err2, string(out2))
		}
		ch := exec.CommandContext(ctx, "git", "-C", dest, "checkout", "--detach", ref)
		ch.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if bout, err3 := ch.CombinedOutput(); err3 != nil {
			_ = os.RemoveAll(dest)
			return "", nil, fmt.Errorf("git checkout %q: %w (%s); first clone attempt: %s", ref, err3, string(bout), string(out))
		}
	} else if cerr != nil {
		return "", nil, fmt.Errorf("git clone: %w (%s)", cerr, string(out))
	}

	sd, err := SecureSubdir(dest, subDirInRepo)
	if err != nil {
		_ = os.RemoveAll(dest)
		return "", nil, err
	}
	if fi, err := os.Stat(sd); err != nil || !fi.IsDir() {
		_ = os.RemoveAll(dest)
		if err != nil {
			return "", nil, fmt.Errorf("scan subdirectory %q: %w", sd, err)
		}
		return "", nil, fmt.Errorf("scan path is not a directory: %s", sd)
	}

	done := false
	cleanupFn := func() {
		if done {
			return
		}
		done = true
		_ = os.RemoveAll(dest)
	}
	return sd, cleanupFn, nil
}
