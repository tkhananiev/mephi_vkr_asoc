package service

import "testing"

func TestVulnerabilityExternalKey(t *testing.T) {
	if got := vulnerabilityExternalKey("CVE-2024-1", "rule-a"); got != "" {
		t.Fatalf("CVE present should clear external key, got %q", got)
	}
	if got := vulnerabilityExternalKey("", "aws-access-key"); got != "aws-access-key" {
		t.Fatalf("want identifier, got %q", got)
	}
}

func TestBuildGroupKeyIncludesExternalKey(t *testing.T) {
	a := buildGroupKey("", "", ".env", "", "aws-access-key")
	b := buildGroupKey("", "", ".env", "", "github-pat")
	if a == b {
		t.Fatalf("distinct secret rules must not share group key: %q", a)
	}
	cveKey := buildGroupKey("CVE-2024-1", "", "pkg", "1.0", "")
	if cveKey != "CVE-2024-1::::pkg::1.0" {
		t.Fatalf("unexpected cve key %q", cveKey)
	}
}
