package models

type TicketRequest struct {
	GroupID        int64  `json:"group_id"`
	GroupKey       string `json:"group_key"`
	Severity       string `json:"severity"`
	AssetsCount    int    `json:"assets_count"`
	CorrelationRef string `json:"correlation_ref"`
}

type TicketResponse struct {
	GroupID         int64  `json:"group_id"`
	JiraIssueKey    string `json:"jira_issue_key"`
	JiraIssueURL    string `json:"jira_issue_url"`
	SyncStatus      string `json:"sync_status"`
	IdempotencyKey  string `json:"idempotency_key"`
}
