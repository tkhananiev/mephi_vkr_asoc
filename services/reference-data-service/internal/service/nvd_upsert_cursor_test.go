package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"mephi_vkr_asoc/services/reference-data-service/internal/models"
)

type upsertFailRepo struct {
	cursor   *models.ReferenceSyncCursor
	upserted []models.ReferenceSyncCursor
	failIDs  map[string]bool
}

func (s *upsertFailRepo) StartSyncRun(context.Context, string) (int64, error) { return 1, nil }
func (s *upsertFailRepo) UpdateSyncRunProgress(context.Context, int64, models.SyncResult) error {
	return nil
}
func (s *upsertFailRepo) FinishSyncRun(context.Context, int64, string, models.SyncResult, *string) error {
	return nil
}
func (s *upsertFailRepo) UpsertRawItem(context.Context, models.RawItem) error { return nil }
func (s *upsertFailRepo) UpsertReferenceRecord(_ context.Context, record models.ReferenceRecord) (bool, error) {
	if s.failIDs[record.ExternalID] {
		return false, errors.New("injected upsert failure")
	}
	return true, nil
}
func (s *upsertFailRepo) InsertReferenceRecordIfAbsent(context.Context, models.ReferenceRecord) (bool, error) {
	return false, nil
}
func (s *upsertFailRepo) ListSyncRuns(context.Context, int) ([]models.SyncRun, error) { return nil, nil }
func (s *upsertFailRepo) CatalogRecordCounts(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *upsertFailRepo) ListRunningSyncRuns(context.Context) ([]models.SyncRun, error) {
	return nil, nil
}
func (s *upsertFailRepo) LastCompletedSyncBySource(context.Context) (map[string]time.Time, error) {
	return nil, nil
}
func (s *upsertFailRepo) GetReferenceSyncCursor(context.Context, string) (*models.ReferenceSyncCursor, error) {
	return s.cursor, nil
}
func (s *upsertFailRepo) UpsertReferenceSyncCursor(_ context.Context, cursor models.ReferenceSyncCursor) error {
	s.upserted = append(s.upserted, cursor)
	cp := cursor
	s.cursor = &cp
	return nil
}
func (s *upsertFailRepo) DeleteReferenceSyncCursor(context.Context, string) error { return nil }

type pageNVD struct {
	page []models.SourceRecord
}

func (p *pageNVD) SyncAllPages(context.Context, func([]models.SourceRecord) error) error {
	return nil
}
func (p *pageNVD) SyncAllPagesModRange(_ context.Context, _, _ time.Time, onPage func([]models.SourceRecord) error) error {
	return onPage(p.page)
}

func TestSyncNVDPaged_WithholdsCursorWhenPageUpsertsFail(t *testing.T) {
	prev := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	repo := &upsertFailRepo{
		cursor: &models.ReferenceSyncCursor{
			SourceCode:           "nvd",
			NVDLastModEnd:        &prev,
			NVDFullSyncCompleted: true,
		},
		failIDs: map[string]bool{"CVE-2026-99999": true},
	}
	paged := &pageNVD{page: []models.SourceRecord{
		{ExternalID: "CVE-2026-00001", Title: "ok"},
		{ExternalID: "CVE-2026-99999", Title: "fail"},
	}}
	svc := &SyncService{repo: repo}

	_, err := svc.syncNVDPaged(context.Background(), paged, 42)
	if err == nil {
		t.Fatal("expected incomplete-page error")
	}
	if len(repo.upserted) != 0 {
		t.Fatalf("cursor must not advance after partial upserts; got %d upserts", len(repo.upserted))
	}
}

func TestSyncNVDPaged_AdvancesCursorWhenPageFullyUpserted(t *testing.T) {
	prev := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	repo := &upsertFailRepo{
		cursor: &models.ReferenceSyncCursor{
			SourceCode:           "nvd",
			NVDLastModEnd:        &prev,
			NVDFullSyncCompleted: true,
		},
		failIDs: map[string]bool{},
	}
	paged := &pageNVD{page: []models.SourceRecord{
		{ExternalID: "CVE-2026-00001", Title: "ok"},
		{ExternalID: "CVE-2026-00002", Title: "ok"},
	}}
	svc := &SyncService{repo: repo}

	result, err := svc.syncNVDPaged(context.Background(), paged, 42)
	if err != nil {
		t.Fatalf("syncNVDPaged: %v", err)
	}
	if result.ItemsProcessed != 2 || result.ItemsDiscovered != 2 {
		t.Fatalf("counts discovered=%d processed=%d", result.ItemsDiscovered, result.ItemsProcessed)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("want 1 cursor upsert, got %d", len(repo.upserted))
	}
}
