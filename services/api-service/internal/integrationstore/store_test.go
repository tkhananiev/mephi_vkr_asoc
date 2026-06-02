package integrationstore

import "testing"

func TestBuiltinCatalogCapabilities(t *testing.T) {
	items := builtinCatalog
	if len(items) < 4 {
		t.Fatalf("expected at least 4 builtin integrations, got %d", len(items))
	}
	byID := make(map[string]Item, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	semgrep, ok := byID["semgrep"]
	if !ok {
		t.Fatal("missing semgrep")
	}
	if semgrep.ScannerName != "semgrep" || semgrep.InputKind != "filesystem" {
		t.Fatalf("semgrep metadata: %+v", semgrep)
	}
	if len(semgrep.Capabilities) == 0 {
		t.Fatal("semgrep capabilities empty")
	}
	sca := byID["trivy-sca"]
	if sca.APIScanPath != "/api/v1/scans/sca" || sca.ScannerName != "trivy-sca" {
		t.Fatalf("trivy-sca: %+v", sca)
	}
	dast := byID["zap-dast"]
	if dast.APIScanPath != "/api/v1/scans/dast" {
		t.Fatalf("zap-dast path: %q", dast.APIScanPath)
	}
}

func TestListMergedIncludesBuiltin(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	merged := s.ListMerged()
	if len(merged) < len(builtinCatalog) {
		t.Fatalf("merged too short: %d", len(merged))
	}
	if merged[0].ID != "semgrep" {
		t.Fatalf("first id: %q", merged[0].ID)
	}
}
