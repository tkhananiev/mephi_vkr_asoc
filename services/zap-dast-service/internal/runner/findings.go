package runner

import "encoding/json"

type normalizedFinding struct {
	AssetID    string         `json:"asset_id"`
	Identifier string         `json:"identifier"`
	Severity   string         `json:"severity"`
	Component  string         `json:"component"`
	Version    string         `json:"version,omitempty"`
	CVE        string         `json:"cve,omitempty"`
	CWE        string         `json:"cwe,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	RawPayload map[string]any `json:"raw_payload,omitempty"`
}

type scanResult struct {
	Findings []normalizedFinding `json:"findings"`
}

func encodeFindings(findings []normalizedFinding) ([]byte, error) {
	if findings == nil {
		findings = []normalizedFinding{}
	}
	return json.Marshal(scanResult{Findings: findings})
}
