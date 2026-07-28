package workspace

import "testing"

func TestSanitizeGitURLForDisplay(t *testing.T) {
	in := "https://ghp_LEAKME_TOKEN:x-oauth-basic@github.com/org/repo.git"
	got := SanitizeGitURLForDisplay(in)
	if got != "https://github.com/org/repo.git" {
		t.Fatalf("got %q", got)
	}
	if SanitizeGitURLForDisplay("git@github.com:org/repo.git") != "git@github.com:org/repo.git" {
		t.Fatal("ssh urls should pass through")
	}
}

func TestRedactGitRemoteInText(t *testing.T) {
	url := "https://ghp_LEAKME_TOKEN@github.com/org/repo.git"
	raw := "fatal: could not read Password for 'https://ghp_LEAKME_TOKEN@github.com/org/repo.git': terminal prompts disabled"
	got := RedactGitRemoteInText(raw, url)
	if stringsContains(got, "ghp_LEAKME_TOKEN") {
		t.Fatalf("token still present: %q", got)
	}
	if !stringsContains(got, "github.com/org/repo.git") {
		t.Fatalf("host/path lost: %q", got)
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
