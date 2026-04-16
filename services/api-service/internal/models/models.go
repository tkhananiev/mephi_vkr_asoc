package models

type ScanRequest struct {
	TargetPath string `json:"target_path"`
	ScannerName string `json:"scanner_name"`
	// SemgrepConfig — путь к YAML внутри контейнера или идентификатор набора правил Semgrep (например p/php, auto).
	// Пусто — значение из APP_SEMGREP_CONFIG.
	SemgrepConfig string `json:"semgrep_config,omitempty"`
}

type SemgrepResult struct {
	Results []SemgrepFinding `json:"results"`
}

type SemgrepFinding struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Extra   struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Metadata struct {
			CWE []string `json:"cwe"`
			CVE string   `json:"cve"`
		} `json:"metadata"`
	} `json:"extra"`
}

type ProcessingIngestRequest struct {
	ScannerName string                  `json:"scanner_name"`
	Findings    []ProcessingFindingItem `json:"findings"`
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

type TicketRequest struct {
	GroupID        int64  `json:"group_id"`
	GroupKey       string `json:"group_key"`
	Severity       string `json:"severity"`
	AssetsCount    int    `json:"assets_count"`
	CorrelationRef string `json:"correlation_ref"`
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
	ScannerName string               `json:"scanner_name"`
	ScanTarget  string               `json:"scan_target"`
	Findings    []ProcessingFindingItem `json:"findings"`
	Processing  ProcessingResponse   `json:"processing"`
	Groups      []GroupResponse      `json:"groups"`
	Tickets     []TicketResponse     `json:"tickets"`
}
