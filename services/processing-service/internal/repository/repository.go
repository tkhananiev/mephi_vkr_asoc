package repository

import (
	"context"

	"mephi_vkr_asoc/services/processing-service/internal/models"
)

type ProcessingRepository interface {
	StartRun(ctx context.Context, sourceName string, findingsReceived int, ownerUserID *int64, channel string, consoleProductID *int64) (int64, error)
	FinishRun(ctx context.Context, runID int64, status string, result models.ProcessingResult, errMsg *string) error
	InsertFinding(ctx context.Context, finding models.Finding) (int64, error)
	FindReferenceRecordIDByCVE(ctx context.Context, cve string) (*int64, error)
	FindReferenceRecordIDByCWE(ctx context.Context, cwe string) (*int64, error)
	// FindCVEAliasByReferenceRecordID — первый алиас CVE у записи каталога (для обогащения строки уязвимости).
	FindCVEAliasByReferenceRecordID(ctx context.Context, referenceRecordID int64) (string, error)
	CreateVulnerability(ctx context.Context, vulnerability models.Vulnerability) (int64, bool, error)
	MergeVulnerabilityCatalog(ctx context.Context, vulnerabilityID int64, cve string, referenceRecordID *int64, correlationStatus string) error
	LinkFindingToVulnerability(ctx context.Context, findingID, vulnerabilityID int64) error
	UpsertGroup(ctx context.Context, groupKey, severity, groupingRule string) (int64, bool, error)
	LinkGroupToVulnerability(ctx context.Context, groupID, vulnerabilityID int64) error
	ListGroups(ctx context.Context, limit int, ownerUserID *int64, consoleProductID *int64) ([]models.VulnerabilityGroup, error)
	ListVulnerabilityReport(ctx context.Context, limit int, ownerUserID *int64, consoleProductID *int64, filter *models.VulnerabilityReportFilter) ([]models.VulnerabilityReportRow, error)
}
