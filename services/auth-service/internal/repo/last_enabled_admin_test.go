package repo

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestErrLastEnabledAdminSentinel(t *testing.T) {
	wrapped := errors.Join(errors.New("ctx"), ErrLastEnabledAdmin)
	if !errors.Is(wrapped, ErrLastEnabledAdmin) {
		t.Fatal("ErrLastEnabledAdmin must be detectable with errors.Is")
	}
}

func TestDeleteAdminGuardsLastEnabled(t *testing.T) {
	src, err := os.ReadFile("repo.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)
	start := strings.Index(code, "func (r *Repo) DeleteAdmin")
	end := strings.Index(code, "func (r *Repo) PromoteConsoleUserToAdmin")
	if start < 0 || end < 0 || end <= start {
		t.Fatal("could not locate DeleteAdmin body")
	}
	body := code[start:end]
	for _, needle := range []string{
		"lockAdminsForUpdate",
		"countEnabledAdminsTx",
		"ErrLastEnabledAdmin",
		"SELECT disabled FROM authn.admins WHERE id = $1",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("DeleteAdmin missing %q", needle)
		}
	}
}

func TestUpdateAdminGuardsLastEnabledDisable(t *testing.T) {
	src, err := os.ReadFile("repo.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)
	start := strings.Index(code, "func (r *Repo) UpdateAdmin")
	end := strings.Index(code, "func (r *Repo) SetAdminPassword")
	if start < 0 || end < 0 || end <= start {
		t.Fatal("could not locate UpdateAdmin body")
	}
	body := code[start:end]
	for _, needle := range []string{
		"disabling := disabled != nil && *disabled",
		"lockAdminsForUpdate",
		"ErrLastEnabledAdmin",
		"countEnabledAdminsTx",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("UpdateAdmin missing %q", needle)
		}
	}
}

func TestDemoteAdminGuardsLastEnabled(t *testing.T) {
	src, err := os.ReadFile("repo.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)
	start := strings.Index(code, "func (r *Repo) DemoteAdminToConsoleUser")
	if start < 0 {
		t.Fatal("could not locate DemoteAdminToConsoleUser")
	}
	body := code[start:]
	for _, needle := range []string{
		"lockAdminsForUpdate",
		"SELECT password_hash, disabled FROM authn.admins WHERE id = $1",
		"ErrLastEnabledAdmin",
		"countEnabledAdminsTx",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("DemoteAdminToConsoleUser missing %q", needle)
		}
	}
}

func TestCountEnabledAdminsQuery(t *testing.T) {
	src, err := os.ReadFile("repo.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `SELECT COUNT(*) FROM authn.admins WHERE disabled = FALSE`) {
		t.Fatal("CountEnabledAdmins must count only enabled admins")
	}
}
