package models

import "time"

type IngestRequest struct {
	ScannerName string       `json:"scanner_name"`
	Findings    []FindingDTO `json:"findings"`
	// OwnerUserID — пользователь консоли (authn.console_users.id); задаётся api-service при JWT role=user.
	OwnerUserID *int64 `json:"owner_user_id,omitempty"`
	// ConsoleProductID — проект консоли (core.console_products.id); только вместе с OwnerUserID, проверяется на api-service.
	ConsoleProductID *int64 `json:"console_product_id,omitempty"`
	// Channel: "manual" | "ci". Пустое значение при приёме на processing трактуется как manual (обратная совместимость).
	Channel string `json:"channel,omitempty"`
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

// VulnerabilityReportFilter — опциональные query-фильтры для GET /api/v1/report/vulnerabilities.
// Строковые поля (кроме run_channel и числовых id): подстрока без учёта регистра.
// run_channel — «ci» / «manual». group_id и vulnerability_id — точное совпадение.
// run_at_from / run_at_to — границы по времени последнего прогона из отчёта (UTC, RFC3339 или YYYY-MM-DD).
type VulnerabilityReportFilter struct {
	GroupKey          *string    `json:"group_key,omitempty"`
	CVE               *string    `json:"cve,omitempty"`
	BDUID             *string    `json:"bdu_id,omitempty"`
	ScannerName       *string    `json:"scanner_name,omitempty"`
	Severity          *string    `json:"severity,omitempty"`
	CatalogSource     *string    `json:"catalog_source,omitempty"`
	RunChannel        *string    `json:"run_channel,omitempty"`
	AssetPath         *string    `json:"asset_path,omitempty"`
	Version           *string    `json:"version,omitempty"`
	CWE               *string    `json:"cwe,omitempty"`
	GroupID           *int64     `json:"group_id,omitempty"`
	VulnerabilityID   *int64     `json:"vulnerability_id,omitempty"`
	RunAtFrom         *time.Time `json:"run_at_from,omitempty"`
	RunAtTo           *time.Time `json:"run_at_to,omitempty"`
}

// VulnerabilityReportRow — одна строка отчёта: одна уязвимость, с привязкой к группе.
type VulnerabilityReportRow struct {
	GroupID         int64      `json:"group_id"`
	GroupKey        string     `json:"group_key"`
	VulnerabilityID int64      `json:"vulnerability_id"`
	CVE             string     `json:"cve"`
	BDUID           string     `json:"bdu_id"`
	ScannerName     string     `json:"scanner_name"`
	AssetPath       string     `json:"asset_path"`
	Version         string     `json:"version"`
	Severity        string     `json:"severity"`
	RunAt           *time.Time `json:"run_at,omitempty"`
	CatalogSource   string     `json:"catalog_source,omitempty"`
	RunChannel      string     `json:"run_channel,omitempty"`
}

// GroupJiraVulnerability — данные для полей задачи Jira по одной уязвимости в группе.
type GroupJiraVulnerability struct {
	VulnerabilityID   int64  `json:"vulnerability_id"`
	AssetPath         string `json:"asset_path"`
	CVE               string `json:"cve"`
	BDUID             string `json:"bdu_id"`
	CVEDescription    string `json:"cve_description"`
	BDUDescription    string `json:"bdu_description"`
	Criticality       string `json:"criticality"`
	CriticalitySource string `json:"criticality_source"`
}

// GroupJiraContext — контекст группы для создания задачи в Jira.
type GroupJiraContext struct {
	GroupID         int64                    `json:"group_id"`
	GroupKey        string                   `json:"group_key"`
	SeverityMax     string                   `json:"severity_max"`
	Vulnerabilities []GroupJiraVulnerability `json:"vulnerabilities"`
}
