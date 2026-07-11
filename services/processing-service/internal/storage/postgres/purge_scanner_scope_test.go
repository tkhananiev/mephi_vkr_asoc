package postgres

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var sqlPlaceholderPattern = regexp.MustCompile(`\$(\d+)`)

func TestPurgeScannerScopeSQLPlaceholdersMatchArguments(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		args int
	}{
		{name: "finding vulnerabilities", sql: deleteScopedFindingVulnerabilitiesSQL, args: 3},
		{name: "findings", sql: deleteScopedFindingsSQL, args: 3},
		{name: "orphan vulnerabilities", sql: deleteOrphanVulnerabilitiesSQL, args: 0},
		{name: "ticket links", sql: deleteEmptyTicketLinksSQL, args: 1},
		{name: "groups", sql: deleteEmptyGroupsSQL, args: 1},
		{name: "processing runs", sql: deleteScopedProcessingRunsSQL, args: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if max := maxSQLPlaceholder(t, tt.sql); max > tt.args {
				t.Fatalf("query references placeholder $%d but Exec supplies %d args", max, tt.args)
			}
		})
	}
}

func TestPurgeScannerScopeUsesExactNullableRunScope(t *testing.T) {
	scopedQueries := []string{
		deleteScopedFindingVulnerabilitiesSQL,
		deleteScopedFindingsSQL,
		deleteScopedProcessingRunsSQL,
	}

	for _, query := range scopedQueries {
		if !strings.Contains(query, "pr.owner_user_id IS NOT DISTINCT FROM $2::bigint") {
			t.Fatalf("owner scope must match NULL exactly, query:\n%s", query)
		}
		if !strings.Contains(query, "pr.console_product_id IS NOT DISTINCT FROM $3::bigint") {
			t.Fatalf("product scope must match NULL exactly, query:\n%s", query)
		}
		if strings.Contains(query, "$2::bigint IS NULL OR pr.owner_user_id") {
			t.Fatalf("nil owner must not widen purge scope to all owners, query:\n%s", query)
		}
		if strings.Contains(query, "$3::bigint IS NULL OR pr.console_product_id") {
			t.Fatalf("nil product must not widen purge scope to all products, query:\n%s", query)
		}
	}
}

func TestPurgeScannerScopeDeletesOnlyGloballyOrphanVulnerabilities(t *testing.T) {
	for _, forbidden := range []string{"processing_runs", "owner_user_id", "console_product_id"} {
		if strings.Contains(deleteOrphanVulnerabilitiesSQL, forbidden) {
			t.Fatalf("orphan vulnerability cleanup must not be scoped by %q:\n%s", forbidden, deleteOrphanVulnerabilitiesSQL)
		}
	}
	if !strings.Contains(deleteOrphanVulnerabilitiesSQL, "WHERE fv.vulnerability_id = v.id") {
		t.Fatalf("orphan vulnerability cleanup must check for any remaining finding link:\n%s", deleteOrphanVulnerabilitiesSQL)
	}
}

func TestPurgeScannerScopeEmptyGroupCleanupDoesNotTreatNilOwnerAsAllUsers(t *testing.T) {
	for _, query := range []string{deleteEmptyTicketLinksSQL, deleteEmptyGroupsSQL} {
		if !strings.Contains(query, "$1::bigint IS NULL AND g.group_key !~ '^u:[0-9]+:'") {
			t.Fatalf("nil owner cleanup must be limited to unowned group keys, query:\n%s", query)
		}
		if strings.Contains(query, "$1::bigint IS NULL\n\t\t\tOR") {
			t.Fatalf("nil owner must not widen empty group cleanup to all users, query:\n%s", query)
		}
	}
}

func maxSQLPlaceholder(t *testing.T, query string) int {
	t.Helper()

	maxPlaceholder := 0
	for _, match := range sqlPlaceholderPattern.FindAllStringSubmatch(query, -1) {
		placeholder, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatalf("invalid SQL placeholder %q: %v", match[0], err)
		}
		if placeholder > maxPlaceholder {
			maxPlaceholder = placeholder
		}
	}
	return maxPlaceholder
}
