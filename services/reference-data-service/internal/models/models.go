package models

import "time"

type SyncRun struct {
	ID              int64      `json:"id"`
	SourceCode      string     `json:"source_code"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	ItemsDiscovered int        `json:"items_discovered"`
	ItemsProcessed  int        `json:"items_processed"`
	ItemsInserted   int        `json:"items_inserted"`
	ItemsUpdated    int        `json:"items_updated"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
}

type RawItem struct {
	SourceCode  string
	ExternalID  string
	SourceURL   string
	ContentType string
	RawPayload  string
	RawHash     string
}

type ReferenceRecord struct {
	ID          int64
	SourceCode  string
	ExternalID  string
	Title       string
	Description string
	Severity    string
	PublishedAt *time.Time
	ModifiedAt  *time.Time
	SourceURL   string
	Status      string
	Metadata    []byte
	Aliases     []ReferenceAlias
}

type ReferenceAlias struct {
	AliasType  string
	AliasValue string
}

type SyncResult struct {
	SourceCode      string `json:"source_code"`
	RunID           int64  `json:"run_id"`
	ItemsDiscovered int    `json:"items_discovered"`
	ItemsProcessed  int    `json:"items_processed"`
	ItemsInserted   int    `json:"items_inserted"`
	ItemsUpdated    int    `json:"items_updated"`
}

type SourceRecord struct {
	ExternalID  string
	Title       string
	Description string
	Severity    string
	PublishedAt *time.Time
	ModifiedAt  *time.Time
	SourceURL   string
	Status      string
	Metadata    []byte
	Aliases     []ReferenceAlias
	RawPayload  string
	ContentType string
}
