package scm

import (
	"context"
	"fmt"
	"net"
	"strings"
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
	resolve := stubResolver(map[string][]string{
		"gitlab.com":     {"172.65.251.78"},
		"internal-host":  {"10.0.0.5"},
		"decimal-local":  {"127.0.0.1"},
		"metadata-alias": {"169.254.169.254"},
	})
	if err := validateGitRemoteURL("https://gitlab.com/org/repo.git", resolve); err != nil {
		t.Fatalf("public gitlab url: %v", err)
	}
	if err := validateGitRemoteURL("git@gitlab.com:org/repo.git", resolve); err != nil {
		t.Fatalf("public git ssh url: %v", err)
	}

	rejected := []string{
		"http://127.0.0.1/repo.git",
		"http://127.1/repo.git",
		"http://2130706433/repo.git",
		"http://0x7f000001/repo.git",
		"http://10.0.0.1/repo.git",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/repo.git",
		"http://metadata.google.internal/computeMetadata/v1",
		"http://internal-host/repo.git",
		"git@internal-host:repo.git",
		"http://metadata-alias/repo.git",
		"http://decimal-local/repo.git",
	}
	for _, raw := range rejected {
		if err := validateGitRemoteURL(raw, resolve); err == nil {
			t.Fatalf("expected rejection for %q", raw)
		}
	}
}

func stubResolver(hosts map[string][]string) ipResolver {
	return func(_ context.Context, host string) ([]net.IPAddr, error) {
		rawAddrs, ok := hosts[strings.ToLower(host)]
		if !ok {
			return nil, fmt.Errorf("unknown host %q", host)
		}
		addrs := make([]net.IPAddr, 0, len(rawAddrs))
		for _, rawAddr := range rawAddrs {
			addrs = append(addrs, net.IPAddr{IP: net.ParseIP(rawAddr)})
		}
		return addrs, nil
	}
}
