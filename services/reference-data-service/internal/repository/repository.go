package repository

import (
	"context"
	"time"

	"mephi_vkr_asoc/services/reference-data-service/internal/models"
)

type ReferenceRepository interface {
	StartSyncRun(ctx context.Context, sourceCode string) (int64, error)
	UpdateSyncRunProgress(ctx context.Context, runID int64, result models.SyncResult) error
	FinishSyncRun(ctx context.Context, runID int64, status string, result models.SyncResult, errMsg *string) error
	UpsertRawItem(ctx context.Context, item models.RawItem) error
	UpsertReferenceRecord(ctx context.Context, record models.ReferenceRecord) (inserted bool, err error)
	// InsertReferenceRecordIfAbsent — только INSERT; при уже существующей паре (source_code, external_id) ничего не меняет (ни поля записи, ни алиасы).
	InsertReferenceRecordIfAbsent(ctx context.Context, record models.ReferenceRecord) (inserted bool, err error)
	ListSyncRuns(ctx context.Context, limit int) ([]models.SyncRun, error)
	CatalogRecordCounts(ctx context.Context) (map[string]int64, error)
	ListRunningSyncRuns(ctx context.Context) ([]models.SyncRun, error)
	LastCompletedSyncBySource(ctx context.Context) (map[string]time.Time, error)
	GetReferenceSyncCursor(ctx context.Context, sourceCode string) (*models.ReferenceSyncCursor, error)
	UpsertReferenceSyncCursor(ctx context.Context, cursor models.ReferenceSyncCursor) error
	DeleteReferenceSyncCursor(ctx context.Context, sourceCode string) error
}
