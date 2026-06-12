package httpapi

import (
	"os/exec"
	"testing"
)

func TestExpandCommandShellQuotesPlaceholders(t *testing.T) {
	body := runRequest{
		TargetPath:       "repo'; echo injected; echo '",
		GitRepositoryURL: "https://example.test/repo.git && echo injected",
		GitRepositoryRef: "main$(echo injected)",
		SemgrepConfig:    "p/java; echo injected",
		ScannerName:      "custom scanner",
		ScannerID:        "runner-1",
	}

	command := expandCommand("printf '<%s>|<%s>|<%s>|<%s>|<%s>|<%s>' {target_path} {git_repository_url} {git_repository_ref} {semgrep_config} {scanner_name} {scanner_id}", body)
	out, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		t.Fatalf("expanded command failed: %v; output=%s", err, out)
	}

	want := "<repo'; echo injected; echo '>|<https://example.test/repo.git && echo injected>|<main$(echo injected)>|<p/java; echo injected>|<custom scanner>|<runner-1>"
	if string(out) != want {
		t.Fatalf("expanded command executed unexpected shell syntax:\nwant %q\n got %q", want, string(out))
	}
}
