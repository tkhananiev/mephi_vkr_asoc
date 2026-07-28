package scm

import "testing"

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
	if err := ValidateGitRemoteURL("https://gitlab.com/org/repo.git"); err != nil {
		t.Fatalf("public gitlab url: %v", err)
	}
	if err := ValidateGitRemoteURL("http://127.0.0.1/repo.git"); err == nil {
		t.Fatal("expected localhost rejection")
	}
}

func TestSanitizeGitURLForDisplay(t *testing.T) {
	in := "https://ghp_LEAKME_TOKEN:x-oauth-basic@github.com/org/repo.git"
	got := SanitizeGitURLForDisplay(in)
	if got != "https://github.com/org/repo.git" {
		t.Fatalf("got %q", got)
	}
}
