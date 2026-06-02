package models

import "encoding/json"

type AdaptResponse struct {
	Findings []FindingItem `json:"findings"`
}

type FindingItem struct {
	AssetID    string         `json:"asset_id"`
	Identifier string         `json:"identifier"`
	Severity   string         `json:"severity"`
	Component  string         `json:"component"`
	Version    string         `json:"version"`
	CVE        string         `json:"cve"`
	CWE        string         `json:"cwe"`
	Metadata   map[string]any   `json:"metadata"`
	RawPayload map[string]any `json:"raw_payload"`
}

type SemgrepResult struct {
	Results []SemgrepFinding `json:"results"`
	Errors  []struct {
		Message string `json:"message"`
		Level   string `json:"level"`
		Type    string `json:"type"`
	} `json:"errors"`
}

type SemgrepFinding struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Extra   struct {
		Message  string          `json:"message"`
		Severity string          `json:"severity"`
		Metadata json.RawMessage `json:"metadata"`
	} `json:"extra"`
}

type GitleaksFinding struct {
	RuleID      string   `json:"RuleID"`
	Description string   `json:"Description"`
	File        string   `json:"File"`
	StartLine   int      `json:"StartLine"`
	Fingerprint string   `json:"Fingerprint"`
	Tags        []string `json:"Tags"`
}
