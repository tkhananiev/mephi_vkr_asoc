package models

import "encoding/json"

type ScanRequest struct {
	TargetPath       string `json:"target_path"`
	TargetURL        string `json:"target_url,omitempty"`
	ScannerName      string `json:"scanner_name"`
	SemgrepConfig    string `json:"semgrep_config,omitempty"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`

	GitRepositoryRef string `json:"git_repository_ref,omitempty"`

	ConsoleProductID *int64 `json:"console_product_id,omitempty"`
}

// Поля цели те же, что у ScanRequest; scanner_id выбирает исполнитель.
type UnifiedScanRequest struct {
	ScannerID        string `json:"scanner_id"`
	TargetPath       string `json:"target_path,omitempty"`
	TargetURL        string `json:"target_url,omitempty"`
	ScannerName      string `json:"scanner_name,omitempty"`
	SemgrepConfig    string `json:"semgrep_config,omitempty"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`
	GitRepositoryRef string `json:"git_repository_ref,omitempty"`
	ConsoleProductID *int64 `json:"console_product_id,omitempty"`

	Options map[string]any `json:"options,omitempty"`
}

// ToScanRequest копирует поля без scanner_id.
func (u UnifiedScanRequest) ToScanRequest() ScanRequest {
	return ScanRequest{
		TargetPath:       u.TargetPath,
		TargetURL:        u.TargetURL,
		ScannerName:      u.ScannerName,
		SemgrepConfig:    u.SemgrepConfig,
		GitRepositoryURL: u.GitRepositoryURL,
		GitRepositoryRef: u.GitRepositoryRef,
		ConsoleProductID: u.ConsoleProductID,
	}
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

type ProcessingIngestRequest struct {
	ScannerName string                  `json:"scanner_name"`
	Findings    []ProcessingFindingItem `json:"findings"`

	OwnerUserID *int64 `json:"owner_user_id,omitempty"`

	ConsoleProductID *int64 `json:"console_product_id,omitempty"`

	Channel string `json:"channel,omitempty"`
}

type ProcessingFindingItem struct {
	AssetID    string         `json:"asset_id"`
	Identifier string         `json:"identifier"`
	Severity   string         `json:"severity"`
	Component  string         `json:"component"`
	Version    string         `json:"version"`
	CVE        string         `json:"cve"`
	CWE        string         `json:"cwe"`
	Metadata   map[string]any `json:"metadata"`
	RawPayload map[string]any `json:"raw_payload"`
}

type ProcessingResponse struct {
	RunID                  int64 `json:"run_id"`
	FindingsReceived       int   `json:"findings_received"`
	FindingsProcessed      int   `json:"findings_processed"`
	VulnerabilitiesCreated int   `json:"vulnerabilities_created"`
	GroupsUpdated          int   `json:"groups_updated"`
}

type IngestAcceptedResponse struct {
	Status         string `json:"status"`
	CorrelationID  string `json:"correlation_id"`
	FindingsQueued int    `json:"findings_queued"`
}

type TicketVulnerabilityDetail struct {
	VulnerabilityID   int64  `json:"vulnerability_id,omitempty"`
	AssetPath         string `json:"asset_path"`
	CVE               string `json:"cve"`
	BDUID             string `json:"bdu_id"`
	CVEDescription    string `json:"cve_description"`
	BDUDescription    string `json:"bdu_description"`
	Criticality       string `json:"criticality"`
	CriticalitySource string `json:"criticality_source"`
}

type TicketRequest struct {
	GroupID         int64                       `json:"group_id"`
	GroupKey        string                      `json:"group_key"`
	Severity        string                      `json:"severity"`
	AssetsCount     int                         `json:"assets_count"`
	CorrelationRef  string                      `json:"correlation_ref"`
	Vulnerabilities []TicketVulnerabilityDetail `json:"vulnerabilities,omitempty"`
}

type GroupJiraContext struct {
	GroupID         int64                       `json:"group_id"`
	GroupKey        string                      `json:"group_key"`
	SeverityMax     string                      `json:"severity_max"`
	Vulnerabilities []TicketVulnerabilityDetail `json:"vulnerabilities"`
}

type TicketResponse struct {
	GroupID        int64  `json:"group_id"`
	JiraIssueKey   string `json:"jira_issue_key"`
	JiraIssueURL   string `json:"jira_issue_url"`
	SyncStatus     string `json:"sync_status"`
	IdempotencyKey string `json:"idempotency_key"`
}

type GroupResponse struct {
	ID           int64  `json:"id"`
	GroupKey     string `json:"group_key"`
	GroupingRule string `json:"grouping_rule"`
	SeverityMax  string `json:"severity_max"`
	AssetsCount  int    `json:"assets_count"`
	Status       string `json:"status"`
}

type PassportResponse struct {
	ScannerName string                  `json:"scanner_name"`
	ScanTarget  string                  `json:"scan_target"`
	Findings    []ProcessingFindingItem `json:"findings"`
	Processing  ProcessingResponse      `json:"processing"`
	Groups      []GroupResponse         `json:"groups"`
	Tickets     []TicketResponse        `json:"tickets"`
}
