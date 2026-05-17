package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mephi_vkr_asoc/services/processing-service/internal/models"
	"mephi_vkr_asoc/services/processing-service/internal/repository"
)

type ProcessingService struct {
	repo repository.ProcessingRepository
}

func New(repo repository.ProcessingRepository) *ProcessingService {
	return &ProcessingService{repo: repo}
}

func (s *ProcessingService) ProcessFindings(ctx context.Context, request models.IngestRequest) (models.ProcessingResult, error) {
	owner := ingestOwner(request)
	runID, err := s.repo.StartRun(ctx, request.ScannerName, len(request.Findings), owner)
	if err != nil {
		return models.ProcessingResult{}, err
	}

	result := models.ProcessingResult{
		RunID:            runID,
		FindingsReceived: len(request.Findings),
	}

	for _, item := range request.Findings {
		refID, correlationStatus := s.resolveCatalogRef(ctx, item)

		effectiveCVE := strings.TrimSpace(item.CVE)
		if effectiveCVE == "" && refID != nil {
			if fromCat, _ := s.repo.FindCVEAliasByReferenceRecordID(ctx, *refID); fromCat != "" {
				effectiveCVE = fromCat
			}
		}

		payload, _ := json.Marshal(map[string]any{
			"metadata":    item.Metadata,
			"raw_payload": item.RawPayload,
		})

		normalizedIdentifier := normalizeIdentifier(item, effectiveCVE)
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

		vulnerabilityID, inserted, err := s.repo.CreateVulnerability(ctx, models.Vulnerability{
			CVEID:              effectiveCVE,
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

		groupKey := scopedGroupKey(owner, buildGroupKey(
			effectiveCVE,
			strings.TrimSpace(item.CWE),
			strings.TrimSpace(item.Component),
			strings.TrimSpace(item.Version),
		))
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

func (s *ProcessingService) ListGroups(ctx context.Context, limit int, ownerUserID *int64) ([]models.VulnerabilityGroup, error) {
	return s.repo.ListGroups(ctx, limit, ownerUserID)
}

func (s *ProcessingService) ListVulnerabilityReport(ctx context.Context, limit int, ownerUserID *int64) ([]models.VulnerabilityReportRow, error) {
	return s.repo.ListVulnerabilityReport(ctx, limit, ownerUserID)
}

func (s *ProcessingService) resolveCatalogRef(ctx context.Context, item models.FindingDTO) (refID *int64, correlationStatus string) {
	correlationStatus = "not_found"
	refID, _ = s.repo.FindReferenceRecordIDByCVE(ctx, strings.TrimSpace(item.CVE))
	if refID != nil {
		return refID, "matched_by_cve"
	}
	refID, _ = s.repo.FindReferenceRecordIDByCWE(ctx, strings.TrimSpace(item.CWE))
	if refID != nil {
		return refID, "matched_by_cwe"
	}
	return nil, "not_found"
}

func normalizeIdentifier(item models.FindingDTO, effectiveCVE string) string {
	if cve := strings.TrimSpace(effectiveCVE); cve != "" {
		return cve
	}
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

func buildGroupKey(effectiveCVE, cwe, component, version string) string {
	parts := []string{
		strings.TrimSpace(effectiveCVE),
		strings.TrimSpace(cwe),
		strings.TrimSpace(component),
		strings.TrimSpace(version),
	}
	return strings.Join(parts, "::")
}

func ingestOwner(request models.IngestRequest) *int64 {
	if request.OwnerUserID != nil && *request.OwnerUserID > 0 {
		return request.OwnerUserID
	}
	return nil
}

func scopedGroupKey(owner *int64, baseKey string) string {
	if owner != nil && *owner > 0 {
		return fmt.Sprintf("u:%d:%s", *owner, baseKey)
	}
	return baseKey
}
