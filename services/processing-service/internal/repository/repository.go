package repository

import (
	"context"

	"mephi_vkr_aspm/services/processing-service/internal/models"
)

type ProcessingRepository interface {
	StartRun(ctx context.Context, sourceName string, findingsReceived int) (int64, error)
	FinishRun(ctx context.Context, runID int64, status string, result models.ProcessingResult, errMsg *string) error
	InsertFinding(ctx context.Context, finding models.Finding) (int64, error)
	FindReferenceRecordIDByCVE(ctx context.Context, cve string) (*int64, error)
	CreateVulnerability(ctx context.Context, vulnerability models.Vulnerability) (int64, bool, error)
	LinkFindingToVulnerability(ctx context.Context, findingID, vulnerabilityID int64) error
	UpsertGroup(ctx context.Context, groupKey, severity, groupingRule string) (int64, bool, error)
	LinkGroupToVulnerability(ctx context.Context, groupID, vulnerabilityID int64) error
	ListGroups(ctx context.Context, limit int) ([]models.VulnerabilityGroup, error)
}
