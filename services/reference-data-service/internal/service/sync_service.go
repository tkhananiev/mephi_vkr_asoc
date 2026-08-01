package service

import (
	"context"
	"fmt"
	"log"
	stdsync "sync"
	"time"

	"mephi_vkr_asoc/services/reference-data-service/internal/kafka"
	"mephi_vkr_asoc/services/reference-data-service/internal/models"
	"mephi_vkr_asoc/services/reference-data-service/internal/repository"
	"mephi_vkr_asoc/services/reference-data-service/internal/source/bdu"
)

type SourceClient interface {
	Fetch(ctx context.Context) ([]models.SourceRecord, error)
}

type NVDSourceClient interface {
	Fetch(ctx context.Context) ([]models.SourceRecord, error)
	FetchByCVE(ctx context.Context, cveID string) ([]models.SourceRecord, error)
}

type NVDFullSync interface {
	SyncAllPages(ctx context.Context, onPage func([]models.SourceRecord) error) error
	SyncAllPagesModRange(ctx context.Context, modStart, modEnd time.Time, onPage func([]models.SourceRecord) error) error
}

const (
	nvdCursorSource       = "nvd"
	nvdLastModMaxWindow   = 120 * 24 * time.Hour
	nvdIncrementalOverlap = 5 * time.Minute
	nvdBetweenWindows     = 700 * time.Millisecond
)

type SyncService struct {
	repo      repository.ReferenceRepository
	publisher kafka.Publisher
	bdu       SourceClient
	nvd       NVDSourceClient
	bduBulk   *bdu.BulkImporter

	nvdGate      stdsync.Mutex // SyncNVD / фоновый HTTP не параллелятся
	bduBulkGate  stdsync.Mutex // полный импорт БДУ
}

func NewSyncService(
	repo repository.ReferenceRepository,
	publisher kafka.Publisher,
	bduClient SourceClient,
	nvdClient NVDSourceClient,
	bduBulk *bdu.BulkImporter,
) *SyncService {
	return &SyncService{
		repo:      repo,
		publisher: publisher,
		bdu:       bduClient,
		nvd:       nvdClient,
		bduBulk:   bduBulk,
	}
}

func (s *SyncService) SyncBDU(ctx context.Context) (models.SyncResult, error) {
	// RSS is a thin incremental feed: never overwrite bulk-imported catalog rows/aliases.
	return s.syncSource(ctx, "bdu_fstec", s.bdu, true)
}

func (s *SyncService) SyncBDUBulk(ctx context.Context) (models.SyncResult, error) {
	s.bduBulkGate.Lock()
	defer s.bduBulkGate.Unlock()
	return s.syncBDUBulkUnlocked(ctx)
}

func (s *SyncService) syncBDUBulkUnlocked(ctx context.Context) (models.SyncResult, error) {
	if s.bduBulk == nil {
		return models.SyncResult{}, fmt.Errorf("bdu bulk import is not configured (enable APP_BDU_BULK_ENABLED; set vulxml via APP_BDU_VULXML_ZIP_PATH or APP_BDU_VULXML_ZIP_URL — path may be .zip or .xml — and xlsx via APP_BDU_VULLIST_XLSX_PATH or APP_BDU_VULLIST_XLSX_URL)")
	}
	runID, err := s.repo.StartSyncRun(ctx, "bdu_fstec")
	if err != nil {
		return models.SyncResult{}, err
	}
	result := models.SyncResult{SourceCode: "bdu_fstec", RunID: runID}
	started := time.Now()
	var batchNum int64
	lastLog := started
	onBatch := func(page []models.SourceRecord) error {
		batchNum++
		d, p, ins, upd := s.applyRecords(ctx, "bdu_fstec", page, true)
		result.ItemsDiscovered += d
		result.ItemsProcessed += p
		result.ItemsInserted += ins
		result.ItemsUpdated += upd
		if err := s.repo.UpdateSyncRunProgress(ctx, runID, result); err != nil {
			return fmt.Errorf("bulk progress: %w", err)
		}
		elapsed := time.Since(started)
		n := len(page)
		if batchNum <= 5 || batchNum%20 == 0 || time.Since(lastLog) >= 90*time.Second {
			lastLog = time.Now()
			log.Printf(
				"[bdu-bulk] run_id=%d elapsed=%s batch=%d размер_батча=%d накопл.: processed=%d inserted=%d updated=%d; "+
					"прогресс в БД: SELECT items_* FROM audit.reference_sync_runs WHERE id=%d;",
				runID, elapsed.Round(time.Second), batchNum, n,
				result.ItemsProcessed, result.ItemsInserted, result.ItemsUpdated, runID,
			)
		}
		return nil
	}
	log.Printf("[bdu-bulk] старт полного импорта run_id=%d (vulxml, затем vullist)", runID)
	if err := s.bduBulk.Import(ctx, onBatch); err != nil {
		errMsg := err.Error()
		_ = s.repo.FinishSyncRun(ctx, runID, "failed", result, &errMsg)
		return result, err
	}
	if err := s.repo.FinishSyncRun(ctx, runID, "completed", result, nil); err != nil {
		return result, fmt.Errorf("finish sync run: %w", err)
	}
	if err := s.publisher.PublishSyncCompleted(ctx, result); err != nil {
		log.Printf("publish sync completed failed: %v", err)
	}
	log.Printf("[bdu-bulk] готово run_id=%d за %v: processed=%d inserted=%d updated=%d",
		runID, time.Since(started).Round(time.Second), result.ItemsProcessed, result.ItemsInserted, result.ItemsUpdated)
	return result, nil
}

func (s *SyncService) SyncBDUBulkAsync() error {
	if !s.bduBulkGate.TryLock() {
		return fmt.Errorf("полный импорт БДУ уже выполняется")
	}
	go func() {
		defer s.bduBulkGate.Unlock()
		bgCtx, cancel := context.WithTimeout(context.Background(), 168*time.Hour)
		defer cancel()
		if _, err := s.syncBDUBulkUnlocked(bgCtx); err != nil {
			log.Printf("SyncBDUBulk async: %v", err)
		}
	}()
	return nil
}

func (s *SyncService) ResetNVDCursor(ctx context.Context) error {
	return s.repo.DeleteReferenceSyncCursor(ctx, nvdCursorSource)
}

func (s *SyncService) syncNVDPaged(ctx context.Context, paged NVDFullSync, runID int64) (models.SyncResult, error) {
	result := models.SyncResult{
		SourceCode: "nvd",
		RunID:      runID,
	}

	pageSeq := 0
	lastProgWrite := time.Time{}
	onPage := func(page []models.SourceRecord) error {
		d, p, ins, upd := s.applyRecords(ctx, "nvd", page, false)
		result.ItemsDiscovered += d
		result.ItemsProcessed += p
		result.ItemsInserted += ins
		result.ItemsUpdated += upd
		pageSeq++
	
		if pageSeq <= 8 || pageSeq%12 == 0 || time.Since(lastProgWrite) >= 4*time.Second {
			lastProgWrite = time.Now()
			if err := s.repo.UpdateSyncRunProgress(ctx, runID, result); err != nil {
				return fmt.Errorf("nvd progress update: %w", err)
			}
		}
		return nil
	}

	cur, err := s.repo.GetReferenceSyncCursor(ctx, nvdCursorSource)
	if err != nil {
		return result, err
	}

	doFull := cur == nil || !cur.NVDFullSyncCompleted
	if cur != nil && cur.NVDFullSyncCompleted && cur.NVDLastModEnd == nil {
		doFull = true
	}

	if !doFull {
		result.SyncMode = "nvd_incremental_lastmodified"
		end := time.Now().UTC()
		last := end.Add(-nvdIncrementalOverlap)
		if cur.NVDLastModEnd != nil {
			last = cur.NVDLastModEnd.UTC().Add(-nvdIncrementalOverlap)
		}
		if last.Before(end) {
			for ws := last; ws.Before(end); {
				we := ws.Add(nvdLastModMaxWindow)
				if we.After(end) {
					we = end
				}
				err := paged.SyncAllPagesModRange(ctx, ws, we, onPage)
				if err != nil {
					return result, err
				}
				if we.Equal(end) || !we.Before(end) {
					break
				}
				ws = we
				select {
				case <-ctx.Done():
					return result, ctx.Err()
				case <-time.After(nvdBetweenWindows):
				}
			}
		}
		now := time.Now().UTC()
		if err := s.repo.UpsertReferenceSyncCursor(ctx, models.ReferenceSyncCursor{
			SourceCode:           nvdCursorSource,
			NVDLastModEnd:        &now,
			NVDFullSyncCompleted: true,
		}); err != nil {
			return result, err
		}
		return result, nil
	}

	result.SyncMode = "nvd_full_catalog_pages"
	err = paged.SyncAllPages(ctx, onPage)
	if err != nil {
		return result, err
	}
	now := time.Now().UTC()
	if err := s.repo.UpsertReferenceSyncCursor(ctx, models.ReferenceSyncCursor{
		SourceCode:           nvdCursorSource,
		NVDLastModEnd:        &now,
		NVDFullSyncCompleted: true,
	}); err != nil {
		return result, err
	}
	return result, nil
}

func (s *SyncService) SyncNVD(ctx context.Context) (models.SyncResult, error) {
	s.nvdGate.Lock()
	defer s.nvdGate.Unlock()
	return s.syncNVDUnlocked(ctx)
}

func (s *SyncService) SyncNVDAsync() error {
	if !s.nvdGate.TryLock() {
		return fmt.Errorf("синхронизация NVD уже выполняется")
	}
	go func() {
		defer s.nvdGate.Unlock()
		bgCtx, cancel := context.WithTimeout(context.Background(), 168*time.Hour)
		defer cancel()
		if _, err := s.syncNVDUnlocked(bgCtx); err != nil {
			log.Printf("SyncNVD async: %v", err)
		}
	}()
	return nil
}

func (s *SyncService) syncNVDUnlocked(ctx context.Context) (models.SyncResult, error) {
	paged, ok := s.nvd.(NVDFullSync)
	if !ok {
		return s.syncSource(ctx, "nvd", s.nvd, false)
	}

	runID, err := s.repo.StartSyncRun(ctx, "nvd")
	if err != nil {
		return models.SyncResult{}, err
	}

	result, err := s.syncNVDPaged(ctx, paged, runID)
	if err != nil {
		errMsg := err.Error()
		_ = s.repo.FinishSyncRun(ctx, runID, "failed", result, &errMsg)
		return result, err
	}

	if err := s.repo.FinishSyncRun(ctx, runID, "completed", result, nil); err != nil {
		return result, fmt.Errorf("finish sync run: %w", err)
	}
	if err := s.publisher.PublishSyncCompleted(ctx, result); err != nil {
		log.Printf("publish sync completed failed: %v", err)
	}
	return result, nil
}

func (s *SyncService) SyncNVDByCVE(ctx context.Context, cveID string) (models.SyncResult, error) {
	if !s.nvdGate.TryLock() {
		return models.SyncResult{}, fmt.Errorf("NVD занят полной или инкрементальной синхронизацией; повторите после её завершения")
	}
	defer s.nvdGate.Unlock()

	runID, err := s.repo.StartSyncRun(ctx, "nvd")
	if err != nil {
		return models.SyncResult{}, err
	}

	result := models.SyncResult{
		SourceCode: "nvd",
		RunID:      runID,
	}

	records, err := s.nvd.FetchByCVE(ctx, cveID)
	if err != nil {
		errMsg := err.Error()
		_ = s.repo.FinishSyncRun(ctx, runID, "failed", result, &errMsg)
		return result, err
	}

	out, err := s.persistRecords(ctx, runID, "nvd", records, false)
	if err != nil {
		return out, err
	}
	out.SyncMode = "nvd_single_cve"
	return out, nil
}

func (s *SyncService) ListRuns(ctx context.Context, limit int) ([]models.SyncRun, error) {
	return s.repo.ListSyncRuns(ctx, limit)
}

func (s *SyncService) CatalogStatus(ctx context.Context) (models.CatalogStatusResponse, error) {
	counts, err := s.repo.CatalogRecordCounts(ctx)
	if err != nil {
		return models.CatalogStatusResponse{}, err
	}
	if counts == nil {
		counts = map[string]int64{}
	}

	running, err := s.repo.ListRunningSyncRuns(ctx)
	if err != nil {
		return models.CatalogStatusResponse{}, err
	}
	if running == nil {
		running = []models.SyncRun{}
	}

	last, err := s.repo.LastCompletedSyncBySource(ctx)
	if err != nil {
		return models.CatalogStatusResponse{}, err
	}
	if last == nil {
		last = map[string]time.Time{}
	}

	var nvdPresent, nvdFull bool
	cur, err := s.repo.GetReferenceSyncCursor(ctx, nvdCursorSource)
	if err != nil {
		return models.CatalogStatusResponse{}, err
	}
	if cur != nil {
		nvdPresent = true
		nvdFull = cur.NVDFullSyncCompleted
	}

	return models.CatalogStatusResponse{
		RecordCounts:     counts,
		RunningSyncs:     running,
		LastCompletedAt:  last,
		NVDCursorPresent: nvdPresent,
		NVDFullSyncDone:  nvdFull,
		SyncInProgress:   len(running) > 0,
	}, nil
}

func (s *SyncService) syncSource(ctx context.Context, sourceCode string, client SourceClient, skipExistingReference bool) (models.SyncResult, error) {
	runID, err := s.repo.StartSyncRun(ctx, sourceCode)
	if err != nil {
		return models.SyncResult{}, err
	}

	result := models.SyncResult{
		SourceCode: sourceCode,
		RunID:      runID,
	}

	records, err := client.Fetch(ctx)
	if err != nil {
		errMsg := err.Error()
		_ = s.repo.FinishSyncRun(ctx, runID, "failed", result, &errMsg)
		return result, err
	}

	return s.persistRecords(ctx, runID, sourceCode, records, skipExistingReference)
}

// Если skipExistingReference=true (BDU RSS и полный bulk): при уже существующей записи каталога только пропуск —
// без UPDATE полей и без DELETE/REPLACE алиасов у старой строки. Нужно, чтобы тонкий RSS не затирал vulxml/xlsx.
func (s *SyncService) applyRecords(ctx context.Context, sourceCode string, records []models.SourceRecord, skipExistingReference bool) (discovered, processed, inserted, updated int) {
	discovered = len(records)
	for _, record := range records {
		rawItem := models.RawItem{
			SourceCode:  sourceCode,
			ExternalID:  record.ExternalID,
			SourceURL:   record.SourceURL,
			ContentType: record.ContentType,
			RawPayload:  record.RawPayload,
		}
		if err := s.repo.UpsertRawItem(ctx, rawItem); err != nil {
			log.Printf("failed to save raw item source=%s external_id=%s: %v", sourceCode, record.ExternalID, err)
			continue
		}

		if skipExistingReference {
			newRow, err := s.repo.InsertReferenceRecordIfAbsent(ctx, models.ReferenceRecord{
				SourceCode:  sourceCode,
				ExternalID:  record.ExternalID,
				Title:       record.Title,
				Description: record.Description,
				Severity:    record.Severity,
				PublishedAt: record.PublishedAt,
				ModifiedAt:  record.ModifiedAt,
				SourceURL:   record.SourceURL,
				Status:      record.Status,
				Metadata:    record.Metadata,
				Aliases:     record.Aliases,
			})
			if err != nil {
				log.Printf("failed to insert-if-absent record source=%s external_id=%s: %v", sourceCode, record.ExternalID, err)
				continue
			}
			processed++
			if newRow {
				inserted++
			}
			continue
		}

		wasNew, err := s.repo.UpsertReferenceRecord(ctx, models.ReferenceRecord{
			SourceCode:  sourceCode,
			ExternalID:  record.ExternalID,
			Title:       record.Title,
			Description: record.Description,
			Severity:    record.Severity,
			PublishedAt: record.PublishedAt,
			ModifiedAt:  record.ModifiedAt,
			SourceURL:   record.SourceURL,
			Status:      record.Status,
			Metadata:    record.Metadata,
			Aliases:     record.Aliases,
		})
		if err != nil {
			log.Printf("failed to upsert record source=%s external_id=%s: %v", sourceCode, record.ExternalID, err)
			continue
		}

		processed++
		if wasNew {
			inserted++
		} else {
			updated++
		}
	}
	return discovered, processed, inserted, updated
}

func (s *SyncService) persistRecords(ctx context.Context, runID int64, sourceCode string, records []models.SourceRecord, skipExistingReference bool) (models.SyncResult, error) {
	result := models.SyncResult{
		SourceCode: sourceCode,
		RunID:      runID,
	}

	d, p, ins, upd := s.applyRecords(ctx, sourceCode, records, skipExistingReference)
	result.ItemsDiscovered = d
	result.ItemsProcessed = p
	result.ItemsInserted = ins
	result.ItemsUpdated = upd

	if err := s.repo.FinishSyncRun(ctx, runID, "completed", result, nil); err != nil {
		return result, fmt.Errorf("finish sync run: %w", err)
	}

	if err := s.publisher.PublishSyncCompleted(ctx, result); err != nil {
		log.Printf("publish sync completed failed: %v", err)
	}

	return result, nil
}
