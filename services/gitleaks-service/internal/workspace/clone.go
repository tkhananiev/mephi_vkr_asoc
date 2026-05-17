package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ValidateGitRemoteURL — допуск только http(s) или ssh git@host:path без file:// и без явного SSRF-хоста.
func ValidateGitRemoteURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty git url")
	}
	if strings.Contains(strings.ToLower(raw), "file://") {
		return fmt.Errorf("file:// remotes not allowed")
	}
	if strings.HasPrefix(strings.ToLower(raw), "git@") {
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
	if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" ||
		strings.HasPrefix(lowerHost, "0.") || lowerHost == "metadata.google.internal" {
		return fmt.Errorf("disallowed host %q", lowerHost)
	}
	return nil
}

// SecureSubdir резолвит относительный подкаталог внутри корня клона без выхода вверх.
func SecureSubdir(repoRoot string, relative string) (string, error) {
	rr := filepath.Clean(repoRoot)
	rel := strings.Trim(strings.TrimSpace(strings.ReplaceAll(relative, "\\", "/")), "/")
	if rel == "" || rel == "." {
		return rr, nil
	}
	part := filepath.Clean(filepath.FromSlash(rel))
	if part == "." {
		return rr, nil
	}
	if strings.Contains(part, "..") || filepath.IsAbs(part) {
		return "", fmt.Errorf("invalid subdirectory %q", relative)
	}
	out := filepath.Clean(filepath.Join(rr, part))
	rrSep := rr + string(filepath.Separator)
	if out != rr && !strings.HasPrefix(out, rrSep) {
		return "", fmt.Errorf("subdirectory escapes clone root")
	}
	return out, nil
}

func randomSuffix() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// PrepareGitWorkspace делает shallow clone в уникальный каталог под workRoot, возвращает каталог скана и удаление.
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
		ch := exec.CommandContext(ctx, "git", "-C", dest, "checkout", ref)
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
