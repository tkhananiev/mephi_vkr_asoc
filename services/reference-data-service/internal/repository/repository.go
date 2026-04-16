package repository

import (
	"context"

	"mephi_vkr_aspm/services/reference-data-service/internal/models"
)

type ReferenceRepository interface {
	StartSyncRun(ctx context.Context, sourceCode string) (int64, error)
	FinishSyncRun(ctx context.Context, runID int64, status string, result models.SyncResult, errMsg *string) error
	UpsertRawItem(ctx context.Context, item models.RawItem) error
	UpsertReferenceRecord(ctx context.Context, record models.ReferenceRecord) (inserted bool, err error)
	ListSyncRuns(ctx context.Context, limit int) ([]models.SyncRun, error)
}
