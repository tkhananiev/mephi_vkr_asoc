package service

import (
	"context"
	"testing"
	"time"

	"mephi_vkr_asoc/services/reference-data-service/internal/models"
)

type stubRefRepo struct {
	cursor       *models.ReferenceSyncCursor
	upserted     []models.ReferenceSyncCursor
	progressCalls int
}

func (s *stubRefRepo) StartSyncRun(context.Context, string) (int64, error) { return 1, nil }
func (s *stubRefRepo) UpdateSyncRunProgress(context.Context, int64, models.SyncResult) error {
	s.progressCalls++
	return nil
}
func (s *stubRefRepo) FinishSyncRun(context.Context, int64, string, models.SyncResult, *string) error {
	return nil
}
func (s *stubRefRepo) UpsertRawItem(context.Context, models.RawItem) error { return nil }
func (s *stubRefRepo) UpsertReferenceRecord(context.Context, models.ReferenceRecord) (bool, error) {
	return false, nil
}
func (s *stubRefRepo) InsertReferenceRecordIfAbsent(context.Context, models.ReferenceRecord) (bool, error) {
	return false, nil
}
func (s *stubRefRepo) ListSyncRuns(context.Context, int) ([]models.SyncRun, error) { return nil, nil }
func (s *stubRefRepo) CatalogRecordCounts(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *stubRefRepo) ListRunningSyncRuns(context.Context) ([]models.SyncRun, error) { return nil, nil }
func (s *stubRefRepo) LastCompletedSyncBySource(context.Context) (map[string]time.Time, error) {
	return nil, nil
}
func (s *stubRefRepo) GetReferenceSyncCursor(context.Context, string) (*models.ReferenceSyncCursor, error) {
	return s.cursor, nil
}
func (s *stubRefRepo) UpsertReferenceSyncCursor(_ context.Context, cursor models.ReferenceSyncCursor) error {
	s.upserted = append(s.upserted, cursor)
	cp := cursor
	s.cursor = &cp
	return nil
}
func (s *stubRefRepo) DeleteReferenceSyncCursor(context.Context, string) error { return nil }

type stubNVDPaged struct {
	calls [][2]time.Time
	delay time.Duration
}

func (s *stubNVDPaged) SyncAllPages(context.Context, func([]models.SourceRecord) error) error {
	return nil
}
func (s *stubNVDPaged) SyncAllPagesModRange(_ context.Context, modStart, modEnd time.Time, onPage func([]models.SourceRecord) error) error {
	s.calls = append(s.calls, [2]time.Time{modStart, modEnd})
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return onPage(nil)
}

func TestSyncNVDPaged_IncrementalCursorUsesCoveredEnd(t *testing.T) {
	prev := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	repo := &stubRefRepo{
		cursor: &models.ReferenceSyncCursor{
			SourceCode:           "nvd",
			NVDLastModEnd:        &prev,
			NVDFullSyncCompleted: true,
		},
	}
	paged := &stubNVDPaged{delay: 20 * time.Millisecond}
	svc := &SyncService{repo: repo}

	before := time.Now().UTC()
	result, err := svc.syncNVDPaged(context.Background(), paged, 7)
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("syncNVDPaged: %v", err)
	}
	if result.SyncMode != "nvd_incremental_lastmodified" {
		t.Fatalf("sync mode: %q", result.SyncMode)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("want 1 cursor upsert, got %d", len(repo.upserted))
	}
	got := repo.upserted[0].NVDLastModEnd
	if got == nil {
		t.Fatal("cursor end is nil")
	}
	if got.After(after) || got.Before(before.Add(-time.Second)) {
		t.Fatalf("cursor end %v outside run window [%v, %v]", got, before, after)
	}
	if len(paged.calls) == 0 {
		t.Fatal("expected mod-range sync call")
	}
	coveredEnd := paged.calls[len(paged.calls)-1][1]
	if !got.Equal(coveredEnd) {
		t.Fatalf("cursor advanced to %v, want covered end %v (not wall-clock after sync)", got, coveredEnd)
	}
}
