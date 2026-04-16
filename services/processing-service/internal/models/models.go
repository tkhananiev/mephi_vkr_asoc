package models

import "time"

type IngestRequest struct {
	ScannerName string       `json:"scanner_name"`
	Findings    []FindingDTO `json:"findings"`
}

type FindingDTO struct {
	AssetID       string                 `json:"asset_id"`
	Identifier    string                 `json:"identifier"`
	Severity      string                 `json:"severity"`
	Component     string                 `json:"component"`
	Version       string                 `json:"version"`
	CVE           string                 `json:"cve"`
	CWE           string                 `json:"cwe"`
	Metadata      map[string]any         `json:"metadata"`
	RawPayload    map[string]any         `json:"raw_payload"`
}

type ProcessingRun struct {
	ID                    int64      `json:"id"`
	SourceName            string     `json:"source_name"`
	Status                string     `json:"status"`
	StartedAt             time.Time  `json:"started_at"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
	FindingsReceived      int        `json:"findings_received"`
	FindingsProcessed     int        `json:"findings_processed"`
	VulnerabilitiesCreated int       `json:"vulnerabilities_created"`
	GroupsUpdated         int        `json:"groups_updated"`
	ErrorMessage          *string    `json:"error_message,omitempty"`
}

type Finding struct {
	ID                   int64
	ProcessingRunID      int64
	ScannerName          string
	AssetID              string
	RawIdentifier        string
	NormalizedIdentifier string
	Severity             string
	Component            string
	Version              string
	PayloadJSON          []byte
}

type Vulnerability struct {
	ID                int64
	CVEID             string
	Product           string
	Version           string
	CWE               string
	NormalizedSeverity string
	CorrelationStatus string
	ReferenceRecordID *int64
}

type VulnerabilityGroup struct {
	ID           int64  `json:"id"`
	GroupKey     string `json:"group_key"`
	GroupingRule string `json:"grouping_rule"`
	SeverityMax  string `json:"severity_max"`
	AssetsCount  int    `json:"assets_count"`
	Status       string `json:"status"`
}

type ProcessingResult struct {
	RunID                  int64 `json:"run_id"`
	FindingsReceived       int   `json:"findings_received"`
	FindingsProcessed      int   `json:"findings_processed"`
	VulnerabilitiesCreated int   `json:"vulnerabilities_created"`
	GroupsUpdated          int   `json:"groups_updated"`
}
