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
	return s.syncSource(ctx, "nvd", s.nvd)
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

func (s *SyncService) persistRecords(ctx context.Context, runID int64, sourceCode string, records []models.SourceRecord) (models.SyncResult, error) {
	result := models.SyncResult{
		SourceCode: sourceCode,
		RunID:      runID,
	}

	result.ItemsDiscovered = len(records)

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

		inserted, err := s.repo.UpsertReferenceRecord(ctx, models.ReferenceRecord{
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

		result.ItemsProcessed++
		if inserted {
			result.ItemsInserted++
		} else {
			result.ItemsUpdated++
		}
	}

	if err := s.repo.FinishSyncRun(ctx, runID, "completed", result, nil); err != nil {
		return result, fmt.Errorf("finish sync run: %w", err)
	}

	if err := s.publisher.PublishSyncCompleted(ctx, result); err != nil {
		log.Printf("publish sync completed failed: %v", err)
	}

	return result, nil
}
