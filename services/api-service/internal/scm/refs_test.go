package scm

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func TestParseLsRemoteHeads(t *testing.T) {
	raw := "abc123\trefs/heads/main\n" +
		"def456\trefs/heads/develop\n" +
		"ghi789\trefs/heads/feature/login\n" +
		"junk line\n" +
		"zzz\trefs/tags/v1.0\n"
	got := parseLsRemoteHeads([]byte(raw))
	want := []string{"develop", "feature/login", "main"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestValidateGitRemoteURL(t *testing.T) {
	stubLookup(t, map[string][]string{
		"git.example.test":      {"93.184.216.34"},
		"internal.example.test": {"10.0.0.12"},
	})

	allowed := []string{
		"https://git.example.test/org/repo.git",
		"https://93.184.216.34/org/repo.git",
		"git@git.example.test:org/repo.git",
		"Git@git.example.test:org/repo.git",
	}
	for _, raw := range allowed {
		if err := ValidateGitRemoteURL(raw); err != nil {
			t.Fatalf("expected %q to be allowed: %v", raw, err)
		}
	}

	rejected := []string{
		"http://127.0.0.1/repo.git",
		"http://169.254.169.254/latest/meta-data/",
		"https://[::1]/repo.git",
		"https://metadata.google.internal/repo.git",
		"https://service.localhost/repo.git",
		"https://internal.example.test/org/repo.git",
		"http://127.1/repo.git",
		"http://2130706433/repo.git",
		"http://0177.0.0.1/repo.git",
		"git@10.0.0.12:org/repo.git",
		"git@[::1]:org/repo.git",
		"file:///tmp/repo",
	}
	for _, raw := range rejected {
		if err := ValidateGitRemoteURL(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func stubLookup(t *testing.T, hosts map[string][]string) {
	t.Helper()
	old := lookupIPAddr
	lookupIPAddr = func(_ context.Context, host string) ([]net.IPAddr, error) {
		rawIPs, ok := hosts[host]
		if !ok {
			return nil, fmt.Errorf("unexpected lookup for %s", host)
		}
		addrs := make([]net.IPAddr, 0, len(rawIPs))
		for _, raw := range rawIPs {
			addrs = append(addrs, net.IPAddr{IP: net.ParseIP(raw)})
		}
		return addrs, nil
	}
	t.Cleanup(func() { lookupIPAddr = old })
}
