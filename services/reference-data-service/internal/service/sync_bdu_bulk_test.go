package service

import (
	"context"
	"testing"
	"time"

	"mephi_vkr_asoc/services/reference-data-service/internal/kafka"
	"mephi_vkr_asoc/services/reference-data-service/internal/models"
)

type recordingCatalogRepo struct {
	upsertCalls       int
	insertAbsentCalls int
	existing          map[string]bool
	upsertAliasCounts map[string]int
}

func (r *recordingCatalogRepo) StartSyncRun(context.Context, string) (int64, error) {
	return 1, nil
}
func (r *recordingCatalogRepo) UpdateSyncRunProgress(context.Context, int64, models.SyncResult) error {
	return nil
}
func (r *recordingCatalogRepo) FinishSyncRun(context.Context, int64, string, models.SyncResult, *string) error {
	return nil
}
func (r *recordingCatalogRepo) UpsertRawItem(context.Context, models.RawItem) error { return nil }
func (r *recordingCatalogRepo) UpsertReferenceRecord(_ context.Context, record models.ReferenceRecord) (bool, error) {
	r.upsertCalls++
	if r.upsertAliasCounts == nil {
		r.upsertAliasCounts = map[string]int{}
	}
	r.upsertAliasCounts[record.ExternalID] = len(record.Aliases)
	wasNew := !r.existing[record.ExternalID]
	r.existing[record.ExternalID] = true
	return wasNew, nil
}
func (r *recordingCatalogRepo) InsertReferenceRecordIfAbsent(_ context.Context, record models.ReferenceRecord) (bool, error) {
	r.insertAbsentCalls++
	if r.existing[record.ExternalID] {
		return false, nil
	}
	r.existing[record.ExternalID] = true
	return true, nil
}
func (r *recordingCatalogRepo) ListSyncRuns(context.Context, int) ([]models.SyncRun, error) {
	return nil, nil
}
func (r *recordingCatalogRepo) CatalogRecordCounts(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (r *recordingCatalogRepo) ListRunningSyncRuns(context.Context) ([]models.SyncRun, error) {
	return nil, nil
}
func (r *recordingCatalogRepo) LastCompletedSyncBySource(context.Context) (map[string]time.Time, error) {
	return nil, nil
}
func (r *recordingCatalogRepo) GetReferenceSyncCursor(context.Context, string) (*models.ReferenceSyncCursor, error) {
	return nil, nil
}
func (r *recordingCatalogRepo) UpsertReferenceSyncCursor(context.Context, models.ReferenceSyncCursor) error {
	return nil
}
func (r *recordingCatalogRepo) DeleteReferenceSyncCursor(context.Context, string) error {
	return nil
}

func TestApplyBDUBulkPageUpgradesThinRSSRows(t *testing.T) {
	repo := &recordingCatalogRepo{
		existing: map[string]bool{
			// Prior scheduled SyncBDU (RSS) inserted a thin catalog row.
			"BDU:2024-00001": true,
		},
	}
	svc := NewSyncService(repo, &kafka.NoopPublisher{}, nil, nil, nil)

	richAliases := []models.ReferenceAlias{
		{AliasType: "CVE", AliasValue: "CVE-2024-0001"},
		{AliasType: "CVE", AliasValue: "CVE-2024-0002"},
		{AliasType: "CWE", AliasValue: "CWE-89"},
	}
	d, p, ins, upd := svc.applyBDUBulkPage(context.Background(), []models.SourceRecord{
		{
			ExternalID:  "BDU:2024-00001",
			Title:       "rich vulxml title",
			Description: "full FSTEC vulxml payload",
			Aliases:     richAliases,
		},
		{
			ExternalID: "BDU:2024-00002",
			Title:      "new from bulk",
			Aliases: []models.ReferenceAlias{
				{AliasType: "CVE", AliasValue: "CVE-2024-1111"},
			},
		},
	})

	if d != 2 || p != 2 {
		t.Fatalf("discovered/processed=%d/%d want 2/2", d, p)
	}
	if repo.insertAbsentCalls != 0 {
		t.Fatalf("bulk must not InsertReferenceRecordIfAbsent (starves thin RSS rows); got %d", repo.insertAbsentCalls)
	}
	if repo.upsertCalls != 2 {
		t.Fatalf("bulk must UpsertReferenceRecord for each item (upgrade thin RSS); got %d upserts", repo.upsertCalls)
	}
	if ins != 1 || upd != 1 {
		t.Fatalf("inserted=%d updated=%d want inserted=1 updated=1", ins, upd)
	}
	if got := repo.upsertAliasCounts["BDU:2024-00001"]; got != len(richAliases) {
		t.Fatalf("thin RSS id must be upgraded with vulxml aliases; got %d want %d", got, len(richAliases))
	}
}
