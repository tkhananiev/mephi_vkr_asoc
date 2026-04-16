package service

import (
	"context"
	"encoding/json"
	"strings"

	"mephi_vkr_aspm/services/processing-service/internal/models"
	"mephi_vkr_aspm/services/processing-service/internal/repository"
)

type ProcessingService struct {
	repo repository.ProcessingRepository
}

func New(repo repository.ProcessingRepository) *ProcessingService {
	return &ProcessingService{repo: repo}
}

func (s *ProcessingService) ProcessFindings(ctx context.Context, request models.IngestRequest) (models.ProcessingResult, error) {
	runID, err := s.repo.StartRun(ctx, request.ScannerName, len(request.Findings))
	if err != nil {
		return models.ProcessingResult{}, err
	}

	result := models.ProcessingResult{
		RunID:            runID,
		FindingsReceived: len(request.Findings),
	}

	for _, item := range request.Findings {
		payload, _ := json.Marshal(map[string]any{
			"metadata":    item.Metadata,
			"raw_payload": item.RawPayload,
		})

		normalizedIdentifier := normalizeIdentifier(item)
		findingID, err := s.repo.InsertFinding(ctx, models.Finding{
			ProcessingRunID:      runID,
			ScannerName:          request.ScannerName,
			AssetID:              item.AssetID,
			RawIdentifier:        item.Identifier,
			NormalizedIdentifier: normalizedIdentifier,
			Severity:             normalizeSeverity(item.Severity),
			Component:            strings.TrimSpace(item.Component),
			Version:              strings.TrimSpace(item.Version),
			PayloadJSON:          payload,
		})
		if err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}

		refID, err := s.repo.FindReferenceRecordIDByCVE(ctx, strings.TrimSpace(item.CVE))
		if err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}

		correlationStatus := "not_found"
		if refID != nil {
			correlationStatus = "matched_by_cve"
		}

		vulnerabilityID, inserted, err := s.repo.CreateVulnerability(ctx, models.Vulnerability{
			CVEID:              strings.TrimSpace(item.CVE),
			Product:            strings.TrimSpace(item.Component),
			Version:            strings.TrimSpace(item.Version),
			CWE:                strings.TrimSpace(item.CWE),
			NormalizedSeverity: normalizeSeverity(item.Severity),
			CorrelationStatus:  correlationStatus,
			ReferenceRecordID:  refID,
		})
		if err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}
		if inserted {
			result.VulnerabilitiesCreated++
		}

		if err := s.repo.LinkFindingToVulnerability(ctx, findingID, vulnerabilityID); err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}

		groupKey := buildGroupKey(item)
		groupID, _, err := s.repo.UpsertGroup(ctx, groupKey, normalizeSeverity(item.Severity), "cve_component_version")
		if err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}
		if err := s.repo.LinkGroupToVulnerability(ctx, groupID, vulnerabilityID); err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}

		result.FindingsProcessed++
		result.GroupsUpdated++
	}

	if err := s.repo.FinishRun(ctx, runID, "completed", result, nil); err != nil {
		return result, err
	}

	return result, nil
}

func (s *ProcessingService) ListGroups(ctx context.Context, limit int) ([]models.VulnerabilityGroup, error) {
	return s.repo.ListGroups(ctx, limit)
}

func normalizeIdentifier(item models.FindingDTO) string {
	if cve := strings.TrimSpace(item.CVE); cve != "" {
		return cve
	}
	return strings.TrimSpace(item.Identifier)
}

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "crit":
		return "critical"
	case "high":
		return "high"
	case "medium", "moderate":
		return "medium"
	case "low":
		return "low"
	default:
		return "unknown"
	}
}

func buildGroupKey(item models.FindingDTO) string {
	parts := []string{
		strings.TrimSpace(item.CVE),
		strings.TrimSpace(item.Component),
		strings.TrimSpace(item.Version),
	}
	return strings.Join(parts, "::")
}
