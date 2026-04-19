package service

import (
	"context"
	"fmt"
	"log"

	"mephi_vkr_aspm/services/reference-data-service/internal/kafka"
	"mephi_vkr_aspm/services/reference-data-service/internal/models"
	"mephi_vkr_aspm/services/reference-data-service/internal/repository"
)

type SourceClient interface {
	Fetch(ctx context.Context) ([]models.SourceRecord, error)
}

type NVDSourceClient interface {
	Fetch(ctx context.Context) ([]models.SourceRecord, error)
	FetchByCVE(ctx context.Context, cveID string) ([]models.SourceRecord, error)
}

// NVDFullSync — постраничная загрузка NVD без хранения всего каталога в памяти.
type NVDFullSync interface {
	SyncAllPages(ctx context.Context, onPage func([]models.SourceRecord) error) error
}

type SyncService struct {
	repo      repository.ReferenceRepository
	publisher kafka.Publisher
	bdu       SourceClient
	nvd       NVDSourceClient
}

func NewSyncService(
	repo repository.ReferenceRepository,
	publisher kafka.Publisher,
	bduClient SourceClient,
	nvdClient NVDSourceClient,
) *SyncService {
	return &SyncService{
		repo:      repo,
		publisher: publisher,
		bdu:       bduClient,
		nvd:       nvdClient,
	}
}

func (s *SyncService) SyncBDU(ctx context.Context) (models.SyncResult, error) {
	return s.syncSource(ctx, "bdu_fstec", s.bdu)
}

func (s *SyncService) SyncNVD(ctx context.Context) (models.SyncResult, error) {
	runID, err := s.repo.StartSyncRun(ctx, "nvd")
	if err != nil {
		return models.SyncResult{}, err
	}

	result := models.SyncResult{
		SourceCode: "nvd",
		RunID:      runID,
	}

	paged, ok := s.nvd.(NVDFullSync)
	if !ok {
		return s.syncSource(ctx, "nvd", s.nvd)
	}

	err = paged.SyncAllPages(ctx, func(page []models.SourceRecord) error {
		d, p, ins, upd := s.applyRecords(ctx, "nvd", page)
		result.ItemsDiscovered += d
		result.ItemsProcessed += p
		result.ItemsInserted += ins
		result.ItemsUpdated += upd
		return nil
	})
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

	return s.persistRecords(ctx, runID, "nvd", records)
}

func (s *SyncService) ListRuns(ctx context.Context, limit int) ([]models.SyncRun, error) {
	return s.repo.ListSyncRuns(ctx, limit)
}

func (s *SyncService) syncSource(ctx context.Context, sourceCode string, client SourceClient) (models.SyncResult, error) {
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

	return s.persistRecords(ctx, runID, sourceCode, records)
}

// applyRecords записывает батч записей; возвращает счётчики для накопления (NVD по страницам).
func (s *SyncService) applyRecords(ctx context.Context, sourceCode string, records []models.SourceRecord) (discovered, processed, inserted, updated int) {
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

func (s *SyncService) persistRecords(ctx context.Context, runID int64, sourceCode string, records []models.SourceRecord) (models.SyncResult, error) {
	result := models.SyncResult{
		SourceCode: sourceCode,
		RunID:      runID,
	}

	d, p, ins, upd := s.applyRecords(ctx, sourceCode, records)
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
