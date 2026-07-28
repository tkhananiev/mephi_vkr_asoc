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

func TestZAP_ExpandsInstances(t *testing.T) {
	raw := []byte(`{
	  "site": [{
	    "@name": "https://example.com",
	    "alerts": [{
	      "pluginid": "10021",
	      "name": "X-Content-Type-Options Header Missing",
	      "riskcode": "1",
	      "cweid": "693",
	      "instances": [
	        {"uri": "https://example.com/a", "method": "GET", "param": ""},
	        {"uri": "https://example.com/b", "method": "GET", "param": "q"}
	      ]
	    }]
	  }]
	}`)
	findings, err := ZAP(raw, "https://example.com")
	if err != nil {
		t.Fatalf("zap: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("want 2 instance findings, got %d", len(findings))
	}
	if findings[0].Component != "https://example.com/a" {
		t.Fatalf("first uri: %q", findings[0].Component)
	}
	if findings[1].Component != "https://example.com/b" {
		t.Fatalf("second uri: %q", findings[1].Component)
	}
	if findings[1].Metadata["param"] != "q" {
		t.Fatalf("param metadata: %#v", findings[1].Metadata["param"])
	}
}

func TestZAP_AlertLevelURIFallback(t *testing.T) {
	raw := []byte(`{
	  "site": [{
	    "alerts": [{
	      "pluginid": "1",
	      "name": "Test",
	      "riskcode": "2",
	      "uri": "https://example.com/only",
	      "param": "id"
	    }]
	  }]
	}`)
	findings, err := ZAP(raw, "https://example.com")
	if err != nil || len(findings) != 1 {
		t.Fatalf("fallback: %v len=%d", err, len(findings))
	}
	if findings[0].Component != "https://example.com/only" {
		t.Fatalf("component: %q", findings[0].Component)
	}
}

