package scm

import (
	"context"
	"net/netip"
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
	publicLookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}
	if err := validateGitRemoteURL("https://gitlab.example/org/repo.git", publicLookup); err != nil {
		t.Fatalf("public gitlab url: %v", err)
	}
	if err := ValidateGitRemoteURL("http://127.0.0.1/repo.git"); err == nil {
		t.Fatal("expected localhost rejection")
	}
}

func TestValidateGitRemoteURLRejectsInternalTargets(t *testing.T) {
	privateLookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.0.0.5")}, nil
	}

	cases := []struct {
		name   string
		raw    string
		lookup lookupNetIPFunc
	}{
		{name: "aws metadata ip", raw: "http://169.254.169.254/latest/meta-data/", lookup: privateLookup},
		{name: "private http ip", raw: "http://10.0.0.1/repo.git", lookup: privateLookup},
		{name: "private ssh ip", raw: "git@10.0.0.1:org/repo.git", lookup: privateLookup},
		{name: "gcp metadata host", raw: "https://metadata.google.internal/computeMetadata/v1/", lookup: privateLookup},
		{name: "private resolved hostname", raw: "https://git.internal.example/repo.git", lookup: privateLookup},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateGitRemoteURL(tc.raw, tc.lookup); err == nil {
				t.Fatalf("expected %q to be rejected", tc.raw)
			}
		})
	}
}
