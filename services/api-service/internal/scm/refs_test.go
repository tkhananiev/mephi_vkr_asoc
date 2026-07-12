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
	lookup := func(_ context.Context, host string) ([]net.IPAddr, error) {
		switch host {
		case "gitlab.com", "github.com":
			return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
		case "internal.example":
			return []net.IPAddr{{IP: net.ParseIP("10.0.0.5")}}, nil
		default:
			return nil, fmt.Errorf("unexpected host %q", host)
		}
	}
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "public https", raw: "https://gitlab.com/org/repo.git"},
		{name: "public git ssh", raw: "git@github.com:org/repo.git"},
		{name: "loopback literal", raw: "http://127.0.0.1/repo.git", wantErr: true},
		{name: "private literal", raw: "https://10.0.0.5/repo.git", wantErr: true},
		{name: "link local metadata literal", raw: "http://169.254.169.254/latest/meta-data/", wantErr: true},
		{name: "metadata hostname", raw: "http://metadata.google.internal/repo.git", wantErr: true},
		{name: "private dns answer", raw: "https://internal.example/repo.git", wantErr: true},
		{name: "ssh loopback literal", raw: "git@127.0.0.1:org/repo.git", wantErr: true},
		{name: "file scheme", raw: "file:///tmp/repo.git", wantErr: true},
		{name: "unsupported scheme", raw: "git://github.com/org/repo.git", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitRemoteURL(context.Background(), tt.raw, lookup)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReadLimited(t *testing.T) {
	got, tooLarge, err := readLimited(strings.NewReader("abcdef"), 6)
	if err != nil {
		t.Fatalf("readLimited exact limit: %v", err)
	}
	if tooLarge {
		t.Fatal("exact limit reported too large")
	}
	if string(got) != "abcdef" {
		t.Fatalf("got %q", string(got))
	}

	got, tooLarge, err = readLimited(strings.NewReader("abcdefg"), 6)
	if err != nil {
		t.Fatalf("readLimited oversized: %v", err)
	}
	if !tooLarge {
		t.Fatal("oversized input was not reported too large")
	}
	if got != nil {
		t.Fatalf("oversized input returned data: %q", string(got))
	}
}
