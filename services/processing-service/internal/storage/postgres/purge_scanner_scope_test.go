package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestPurgeScannerScopeSQLDoesNotUseNullAsWildcard(t *testing.T) {
	src, err := os.ReadFile("purge_scanner_scope.go")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(src)

	for _, forbidden := range []string{
		"$2::bigint IS NULL OR pr.owner_user_id",
		"$3::bigint IS NULL OR pr.console_product_id",
		"$2::bigint IS NULL OR pr_keep.owner_user_id",
		"$3::bigint IS NULL OR pr_keep.console_product_id",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("purge SQL still treats NULL scope as wildcard: %q", forbidden)
		}
	}
	for _, required := range []string{
		"pr.owner_user_id IS NOT DISTINCT FROM $2::bigint",
		"pr.console_product_id IS NOT DISTINCT FROM $3::bigint",
		"g.group_key NOT LIKE 'u:%'",
		"pr_keep.owner_user_id IS NOT DISTINCT FROM $2::bigint",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("purge SQL missing exact-scope predicate: %q", required)
		}
	}
}

func TestPurgeScannerScopeDeletesOnlyGloballyOrphanedVulnerabilities(t *testing.T) {
	src, err := os.ReadFile("purge_scanner_scope.go")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(src)

	vulnerabilityDelete := strings.Split(sql, "DELETE FROM core.vulnerabilities v")
	if len(vulnerabilityDelete) < 2 {
		t.Fatal("missing vulnerability orphan cleanup")
	}
	cleanupBlock := strings.Split(vulnerabilityDelete[1], "DELETE FROM integration.ticket_links tl")[0]
	if strings.Contains(cleanupBlock, "processing_runs pr") {
		t.Fatalf("vulnerability cleanup must not be limited to the purged scope:\n%s", cleanupBlock)
	}
	if !strings.Contains(cleanupBlock, "WHERE fv.vulnerability_id = v.id") {
		t.Fatalf("vulnerability cleanup must check global finding references:\n%s", cleanupBlock)
	}
}
