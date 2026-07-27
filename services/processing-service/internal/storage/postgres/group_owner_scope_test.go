package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestUpdateGroupStatusScopesByGroupKeyNotSharedVuln(t *testing.T) {
	if !strings.Contains(updateGroupStatusSQL, "g.group_key LIKE ('u:' || $3::bigint::text || ':%')") {
		t.Fatal("UpdateGroupStatus must require owner group_key prefix when scoped")
	}
	for _, forbidden := range []string{
		"finding_vulnerabilities",
		"processing_runs",
		"pr.owner_user_id",
	} {
		if strings.Contains(updateGroupStatusSQL, forbidden) {
			t.Fatalf("UpdateGroupStatus must not authorize via shared vuln/finding links (%q)", forbidden)
		}
	}
}

func TestListVulnerabilityReportScopesGroupsByOwnerKey(t *testing.T) {
	src, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("read repository.go: %v", err)
	}
	body := string(src)
	fnStart := strings.Index(body, "func (r *Repository) ListVulnerabilityReport")
	if fnStart < 0 {
		t.Fatal("ListVulnerabilityReport not found")
	}
	fnBody := body[fnStart:]
	if end := strings.Index(fnBody, "\nfunc (r *Repository) GetGroupJiraContext"); end > 0 {
		fnBody = fnBody[:end]
	}
	if !strings.Contains(fnBody, listVulnerabilityReportOwnerGroupPredicate) {
		t.Fatal("ListVulnerabilityReport must filter groups by owner group_key prefix")
	}
	// Ensure we do not rely solely on vulnerability-linked findings for group visibility.
	if strings.Count(fnBody, "g.group_key LIKE") < 1 {
		t.Fatal("expected group_key owner predicate in report query")
	}
}
