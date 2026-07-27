package workspace

import (
	"strings"
	"testing"
)

func TestGitHTTPSafeArgsDisableRedirects(t *testing.T) {
	t.Parallel()
	args := gitHTTPSafeArgs()
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "http.followRedirects=false") {
		t.Fatalf("expected http.followRedirects=false in %v", args)
	}
	if !strings.Contains(joined, "protocol.file.allow=never") {
		t.Fatalf("expected protocol.file.allow=never in %v", args)
	}
}
