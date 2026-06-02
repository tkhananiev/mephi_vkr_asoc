package models

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

type TicketResponse struct {
	GroupID         int64  `json:"group_id"`
	JiraIssueKey    string `json:"jira_issue_key"`
	JiraIssueURL    string `json:"jira_issue_url"`
	SyncStatus      string `json:"sync_status"`
	IdempotencyKey  string `json:"idempotency_key"`
}
