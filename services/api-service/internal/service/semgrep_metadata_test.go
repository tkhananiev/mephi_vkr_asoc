package service

import (
	"encoding/json"
	"testing"
)

func TestSemgrepCWEFromMetadataString(t *testing.T) {
	meta := json.RawMessage(`{"cwe":"CWE-79: XSS"}`)
	if got := semgrepCWEFromMetadata(meta); got != "CWE-79" {
		t.Fatalf("got %q", got)
	}
}

func TestSemgrepCWEFromMetadataArray(t *testing.T) {
	meta := json.RawMessage(`{"cwe":["CWE-89: SQL Injection"]}`)
	if got := semgrepCWEFromMetadata(meta); got != "CWE-89" {
		t.Fatalf("got %q", got)
	}
}

func TestSemgrepCWEFromMetadataBareDigits(t *testing.T) {
	meta := json.RawMessage(`{"cwe":"917"}`)
	if got := semgrepCWEFromMetadata(meta); got != "CWE-917" {
		t.Fatalf("got %q", got)
	}
}

func TestSemgrepCVEFromMetadataReferences(t *testing.T) {
	meta := json.RawMessage(`{"references":["https://cwe.mitre.org/data/definitions/89.html","https://nvd.nist.gov/vuln/detail/cve-2021-44228"]}`)
	if got := semgrepCVEFromMetadata(meta); got != "CVE-2021-44228" {
		t.Fatalf("got %q", got)
	}
}
