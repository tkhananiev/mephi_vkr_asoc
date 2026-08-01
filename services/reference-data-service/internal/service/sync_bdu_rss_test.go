package service

import (
	"context"
	"testing"
	"time"

	"mephi_vkr_asoc/services/reference-data-service/internal/kafka"
	"mephi_vkr_asoc/services/reference-data-service/internal/models"
)

type stubBDUClient struct {
	records []models.SourceRecord
}

func (c stubBDUClient) Fetch(context.Context) ([]models.SourceRecord, error) {
	return c.records, nil
}

// recordingRepo tracks which catalog write path SyncBDU uses.
type recordingRepo struct {
	upsertCalls       int
	insertAbsentCalls int
	existing          map[string]bool
}

func (r *recordingRepo) StartSyncRun(context.Context, string) (int64, error) { return 1, nil }
func (r *recordingRepo) UpdateSyncRunProgress(context.Context, int64, models.SyncResult) error {
	return nil
}
func (r *recordingRepo) FinishSyncRun(context.Context, int64, string, models.SyncResult, *string) error {
	return nil
}
func (r *recordingRepo) UpsertRawItem(context.Context, models.RawItem) error { return nil }
func (r *recordingRepo) UpsertReferenceRecord(_ context.Context, record models.ReferenceRecord) (bool, error) {
	r.upsertCalls++
	return !r.existing[record.ExternalID], nil
}
func (r *recordingRepo) InsertReferenceRecordIfAbsent(_ context.Context, record models.ReferenceRecord) (bool, error) {
	r.insertAbsentCalls++
	if r.existing[record.ExternalID] {
		return false, nil
	}
	r.existing[record.ExternalID] = true
	return true, nil
}
func (r *recordingRepo) ListSyncRuns(context.Context, int) ([]models.SyncRun, error) { return nil, nil }
func (r *recordingRepo) CatalogRecordCounts(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (r *recordingRepo) ListRunningSyncRuns(context.Context) ([]models.SyncRun, error) {
	return nil, nil
}
func (r *recordingRepo) LastCompletedSyncBySource(context.Context) (map[string]time.Time, error) {
	return nil, nil
}
func (r *recordingRepo) GetReferenceSyncCursor(context.Context, string) (*models.ReferenceSyncCursor, error) {
	return nil, nil
}
func (r *recordingRepo) UpsertReferenceSyncCursor(context.Context, models.ReferenceSyncCursor) error {
	return nil
}
func (r *recordingRepo) DeleteReferenceSyncCursor(context.Context, string) error { return nil }

func TestSyncBDUSkipsExistingCatalogRows(t *testing.T) {
	repo := &recordingRepo{
		existing: map[string]bool{
			// Simulates a prior bulk import with rich aliases already stored.
			"BDU:2024-00001": true,
		},
	}
	client := stubBDUClient{records: []models.SourceRecord{
		{
			ExternalID:  "BDU:2024-00001",
			Title:       "thin RSS title",
			Description: "RSS often has fewer CVE aliases than vulxml bulk",
			Aliases: []models.ReferenceAlias{
				{AliasType: "CVE", AliasValue: "CVE-2024-0001"},
			},
		},
		{
			ExternalID: "BDU:2024-99999",
			Title:      "brand new from RSS",
			Aliases: []models.ReferenceAlias{
				{AliasType: "CVE", AliasValue: "CVE-2024-99999"},
			},
		},
	}}

	svc := NewSyncService(repo, &kafka.NoopPublisher{}, client, nil, nil)
	result, err := svc.SyncBDU(context.Background())
	if err != nil {
		t.Fatalf("SyncBDU: %v", err)
	}
	if repo.upsertCalls != 0 {
		t.Fatalf("RSS SyncBDU must not UpsertReferenceRecord (would wipe bulk aliases); got %d upserts", repo.upsertCalls)
	}
	if repo.insertAbsentCalls != 2 {
		t.Fatalf("expected InsertReferenceRecordIfAbsent for each RSS item, got %d", repo.insertAbsentCalls)
	}
	if result.ItemsInserted != 1 {
		t.Fatalf("expected 1 insert (new id only), got inserted=%d updated=%d", result.ItemsInserted, result.ItemsUpdated)
	}
	if result.ItemsUpdated != 0 {
		t.Fatalf("existing bulk row must not count as updated, got updated=%d", result.ItemsUpdated)
	}
}
