package adapt

import (
	"testing"
)

func TestTrivy(t *testing.T) {
	raw := []byte(`{
	  "Results": [{
	    "Target": "go.mod",
	    "Vulnerabilities": [{
	      "VulnerabilityID": "CVE-2024-0001",
	      "PkgName": "example/pkg",
	      "InstalledVersion": "1.0.0",
	      "Severity": "HIGH",
	      "Title": "Test vuln"
	    }]
	  }]
	}`)
	findings, err := Trivy(raw)
	if err != nil {
		t.Fatalf("trivy: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].CVE != "CVE-2024-0001" {
		t.Fatalf("cve: %q", findings[0].CVE)
	}
}

func TestFlexible_TrivyAndFindings(t *testing.T) {
	raw := []byte(`{"Results":[{"Target":"pom.xml","Vulnerabilities":[{"VulnerabilityID":"CVE-2023-9","PkgName":"lib","Severity":"MEDIUM"}]}]}`)
	findings, err := Flexible(raw, "")
	if err != nil || len(findings) != 1 {
		t.Fatalf("trivy flexible: %v len=%d", err, len(findings))
	}
	raw2 := []byte(`{"findings":[{"asset_id":"host","identifier":"dast-http-500","severity":"high","component":"https://example.com"}]}`)
	findings2, err := Flexible(raw2, "")
	if err != nil || len(findings2) != 1 || findings2[0].Identifier != "dast-http-500" {
		t.Fatalf("findings wrap: %v %+v", err, findings2)
	}
}

func TestFlexible_GitleaksNativeArrayNotBlank(t *testing.T) {
	raw := []byte(`[
	  {
	    "RuleID": "aws-access-key",
	    "Description": "AWS Access Key",
	    "File": "config/secrets.env",
	    "StartLine": 12,
	    "Fingerprint": "abc123",
	    "Tags": ["key","aws"]
	  }
	]`)
	findings, err := Flexible(raw, "")
	if err != nil {
		t.Fatalf("flexible gitleaks: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Identifier != "aws-access-key" {
		t.Fatalf("identifier: %q", findings[0].Identifier)
	}
	if findings[0].AssetID != "secrets.env" {
		t.Fatalf("asset_id: %q", findings[0].AssetID)
	}
	if findings[0].Severity != "high" {
		t.Fatalf("severity: %q", findings[0].Severity)
	}
}

func TestFlexible_GitleaksWrappedUnderFindingsNotBlank(t *testing.T) {
	raw := []byte(`{"findings":[{
	  "RuleID": "generic-api-key",
	  "Description": "API Key",
	  "File": "app/config.yaml",
	  "StartLine": 4,
	  "Fingerprint": "fp-1"
	}]}`)
	findings, err := Flexible(raw, "")
	if err != nil {
		t.Fatalf("flexible wrapped gitleaks: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Identifier != "generic-api-key" {
		t.Fatalf("identifier: %q (blank FindingItem fallthrough failed)", findings[0].Identifier)
	}
	if findings[0].AssetID != "config.yaml" {
		t.Fatalf("asset_id: %q", findings[0].AssetID)
	}
}

func TestFlexible_NormalizedArrayStillAccepted(t *testing.T) {
	raw := []byte(`[{"asset_id":"app","identifier":"custom-rule","severity":"medium","component":"main.go"}]`)
	findings, err := Flexible(raw, "")
	if err != nil || len(findings) != 1 || findings[0].Identifier != "custom-rule" {
		t.Fatalf("normalized array: %v %+v", err, findings)
	}
}
