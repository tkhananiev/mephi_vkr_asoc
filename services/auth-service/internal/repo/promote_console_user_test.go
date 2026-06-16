package repo

import (
	"os"
	"strings"
	"testing"
)

func TestPromoteConsoleUserToAdminGuardsProductCascade(t *testing.T) {
	src, err := os.ReadFile("repo.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)
	start := strings.Index(code, "func (r *Repo) PromoteConsoleUserToAdmin")
	end := strings.Index(code, "func (r *Repo) DemoteAdminToConsoleUser")
	if start < 0 || end < 0 || end <= start {
		t.Fatal("could not locate PromoteConsoleUserToAdmin body")
	}
	code = code[start:end]

	lockIdx := strings.Index(code, "SELECT password_hash FROM authn.console_users WHERE id = $1 FOR UPDATE")
	countIdx := strings.Index(code, "SELECT COUNT(*) FROM core.console_products WHERE owner_user_id = $1")
	errIdx := strings.Index(code, "ErrConsoleUserHasProducts")
	deleteIdx := strings.Index(code, "DELETE FROM authn.console_users WHERE id = $1")
	for name, idx := range map[string]int{
		"user lock":           lockIdx,
		"product count":       countIdx,
		"product error":       errIdx,
		"console user delete": deleteIdx,
	} {
		if idx < 0 {
			t.Fatalf("missing %s guard element", name)
		}
	}
	if !(lockIdx < countIdx && countIdx < deleteIdx && errIdx < deleteIdx) {
		t.Fatalf("product guard must run before deleting console user; lock=%d count=%d err=%d delete=%d", lockIdx, countIdx, errIdx, deleteIdx)
	}
}
