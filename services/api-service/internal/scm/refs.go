package scm

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

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

// SanitizeGitURLForDisplay strips userinfo (tokens/passwords) from http(s) remotes.
func SanitizeGitURLForDisplay(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = nil
	return u.String()
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
