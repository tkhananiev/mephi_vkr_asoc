package httpapi

import (
	"os"
	"strings"
	"testing"
)

func TestHandlersMapLastEnabledAdminConflicts(t *testing.T) {
	src, err := os.ReadFile("handlers.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)
	for _, needle := range []string{
		`errors.Is(err, repo.ErrLastEnabledAdmin)`,
		`cannot disable the last enabled administrator`,
		`cannot delete the last enabled administrator`,
		`cannot demote the last enabled administrator`,
	} {
		if !strings.Contains(code, needle) {
			t.Fatalf("handlers.go missing %q", needle)
		}
	}
	// Pre-checks that counted disabled rows must not remain.
	if strings.Contains(code, "h.r.AdminCount(") {
		t.Fatal("handlers must not use AdminCount for last-admin guards (counts disabled rows)")
	}
}
