package runner

import (
	"testing"
)

func TestFindingsFromZAPReport(t *testing.T) {
	raw := []byte(`{
  "site": [{
    "@name": "https://example.com",
    "alerts": [{
      "pluginid": "10021",
      "alert": "X-Content-Type-Options Header Missing",
      "name": "X-Content-Type-Options Header Missing",
      "riskcode": "1",
      "confidence": "2",
      "desc": "Missing header",
      "uri": "https://example.com/",
      "cweid": "693"
    }]
  }]
}`)
	findings, err := findingsFromZAPReport(raw, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Identifier != "zap-10021" {
		t.Fatalf("identifier: %q", f.Identifier)
	}
	if f.Severity != "low" {
		t.Fatalf("severity: %q", f.Severity)
	}
	if f.CWE != "CWE-693" {
		t.Fatalf("cwe: %q", f.CWE)
	}
	if f.Metadata["engine"] != "owasp-zap" {
		t.Fatalf("engine: %v", f.Metadata["engine"])
	}
}

func TestZAPRiskToSeverity(t *testing.T) {
	cases := map[string]string{
		"3": "high",
		"2": "medium",
		"1": "low",
		"0": "info",
	}
	for in, want := range cases {
		if got := zapRiskToSeverity(in); got != want {
			t.Fatalf("%s: got %q want %q", in, got, want)
		}
	}
}
