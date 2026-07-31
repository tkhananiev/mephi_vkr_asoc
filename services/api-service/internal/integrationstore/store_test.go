package integrationstore

import (
	"encoding/json"
	"strings"
	"testing"
)

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

func TestListPublicOmitsExecutionSecrets(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	secretURL := "https://token:s3cret@runner.internal/api/v1/run"
	secretCmd := "scanner --token leaktoken --path {target_path}"
	if err := s.ReplaceAdditional([]Item{{
		ID:                    "custom-scanner",
		Kind:                  "SAST",
		Title:                 "Custom",
		Summary:               "Custom scanner",
		Phase:                 "ready",
		Enabled:               true,
		InputKind:             "filesystem",
		ScannerName:           "custom-scanner",
		ScannerInvokeURL:      "https://runner.internal/api/v1/run",
		RunnerCommand:         secretCmd,
		InvokePayloadTemplate: `{"auth":"leaktoken"}`,
		NetworkIP:             "10.0.0.5",
		NetworkHostname:       "runner.internal",
		NetworkPort:           "8080",
		InvokeHint:            "Use product SCM context",
	}}); err != nil {
		t.Fatal(err)
	}

	public := s.ListPublic()
	var found *PublicItem
	for i := range public {
		if public[i].ID == "custom-scanner" {
			found = &public[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected custom-scanner in public catalog")
	}
	if found.Title != "Custom" || found.ScannerName != "custom-scanner" {
		t.Fatalf("public metadata: %+v", found)
	}
	if found.InvokeHint != "Use product SCM context" {
		t.Fatalf("invoke_hint: %q", found.InvokeHint)
	}

	raw, err := json.Marshal(found)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, leak := range []string{
		"scanner_invoke_url",
		"runner_command",
		"invoke_payload_template",
		"network_ip",
		"network_hostname",
		"network_port",
		"network_host",
		secretURL,
		"leaktoken",
		"10.0.0.5",
		"runner.internal",
	} {
		if strings.Contains(body, leak) {
			t.Fatalf("public JSON leaked %q: %s", leak, body)
		}
	}

	admin := s.AdminList()
	var adminItem *Item
	for i := range admin {
		if admin[i].Source == "additional" && admin[i].Integration.ID == "custom-scanner" {
			adminItem = &admin[i].Integration
			break
		}
	}
	if adminItem == nil {
		t.Fatal("expected admin additional entry")
	}
	if adminItem.RunnerCommand != secretCmd || adminItem.ScannerInvokeURL == "" {
		t.Fatalf("admin view should retain secrets: %+v", adminItem)
	}
}

func TestValidateRejectsInvokeURLUserinfo(t *testing.T) {
	err := Validate(Item{
		ID:               "x",
		Kind:             "SAST",
		Title:            "X",
		Phase:            "ready",
		InputKind:        "filesystem",
		ScannerName:      "x",
		ScannerInvokeURL: "https://user:pass@runner.example/run",
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "credentials") {
		t.Fatalf("expected credentials rejection, got %v", err)
	}
}
