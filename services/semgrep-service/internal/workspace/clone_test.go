package workspace

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestSecureSubdirRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("token"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := SecureSubdir(root, "escape"); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}

func TestSecureSubdirAllowsInRepoPath(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "src")
	if err := os.Mkdir(sub, 0750); err != nil {
		t.Fatal(err)
	}
	got, err := SecureSubdir(root, "src")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != resolved {
		t.Fatalf("got %q want %q", got, resolved)
	}
}

func TestAssertPathUnderRootRejectsEtc(t *testing.T) {
	root := t.TempDir()
	if _, err := AssertPathUnderRoot("/etc", root); err == nil {
		t.Fatal("expected /etc to be rejected")
	}
}

func TestValidateGitRemoteURLRejectsPrivateAndMetadata(t *testing.T) {
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("10.0.0.8")}}, nil
	}
	cases := []string{
		"http://169.254.169.254/latest/meta-data",
		"https://10.1.2.3/repo.git",
		"http://127.0.0.1/repo.git",
		"git@127.0.0.1:repo.git",
	}
	for _, raw := range cases {
		if err := validateGitRemoteURL(context.Background(), raw, lookup); err == nil {
			t.Fatalf("expected reject for %q", raw)
		}
	}
	if err := validateGitRemoteURL(context.Background(), "https://gitlab.com/org/repo.git", func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("1.2.3.4")}}, nil
	}); err != nil {
		t.Fatalf("public remote rejected: %v", err)
	}
}
