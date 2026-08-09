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
	scanner := strings.TrimSpace(request.ScannerName)
	if scanner == "" {
		return models.ProcessingResult{}, fmt.Errorf("scanner_name is required")
	}
	if err := s.repo.PurgeScannerScope(ctx, scanner, owner, request.ConsoleProductID); err != nil {
		return models.ProcessingResult{}, fmt.Errorf("purge prior scanner data: %w", err)
	}
	runID, err := s.repo.StartRun(ctx, scanner, len(request.Findings), owner, normalizeRunChannel(request.Channel), request.ConsoleProductID)
	if err != nil {
		return models.ProcessingResult{}, err
	}

	result := models.ProcessingResult{
		RunID:            runID,
		FindingsReceived: len(request.Findings),
	}

	for _, raw := range request.Findings {
		item := enrichFindingCatalogFields(raw)
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
			ScannerName:          scanner,
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
		if err := s.repo.MergeVulnerabilityCatalog(ctx, vulnerabilityID, effectiveCVE, refID, correlationStatus); err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}

		if err := s.repo.LinkFindingToVulnerability(ctx, findingID, vulnerabilityID); err != nil {
			errMsg := err.Error()
			_ = s.repo.FinishRun(ctx, runID, "failed", result, &errMsg)
			return result, err
		}

		groupKey := scopedGroupKey(owner, request.ConsoleProductID, buildGroupKey(
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

func (s *ProcessingService) ListGroups(ctx context.Context, limit int, ownerUserID *int64, consoleProductID *int64, statusFilter string) ([]models.VulnerabilityGroup, error) {
	return s.repo.ListGroups(ctx, limit, ownerUserID, consoleProductID, statusFilter)
}

var allowedGroupStatuses = map[string]struct{}{
	"open":            {},
	"false_positive":  {},
	"risk_accepted":   {},
}

func (s *ProcessingService) UpdateGroupStatus(ctx context.Context, groupID int64, status string, ownerUserID *int64, consoleProductID *int64) (models.VulnerabilityGroup, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if _, ok := allowedGroupStatuses[normalized]; !ok {
		return models.VulnerabilityGroup{}, fmt.Errorf("invalid status: allowed open, false_positive, risk_accepted")
	}
	if groupID <= 0 {
		return models.VulnerabilityGroup{}, fmt.Errorf("invalid group id")
	}
	return s.repo.UpdateGroupStatus(ctx, groupID, normalized, ownerUserID, consoleProductID)
}

func (s *ProcessingService) ListVulnerabilityReport(ctx context.Context, limit int, ownerUserID *int64, consoleProductID *int64, filter *models.VulnerabilityReportFilter) ([]models.VulnerabilityReportRow, error) {
	return s.repo.ListVulnerabilityReport(ctx, limit, ownerUserID, consoleProductID, filter)
}

func (s *ProcessingService) GetGroupJiraContext(ctx context.Context, groupID int64, ownerUserID *int64, consoleProductID *int64) (models.GroupJiraContext, error) {
	return s.repo.GetGroupJiraContext(ctx, groupID, ownerUserID, consoleProductID)
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

func normalizeRunChannel(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), "ci") {
		return "ci"
	}
	return "manual"
}

// scopedGroupKey isolates triage/Jira groups per console owner and product.
// Without a product segment, two products owned by the same user that share a
// CVE/component signature collapse onto one group_key — false_positive on
// product A also hides the finding for product B, and a shared ticket_links row
// is overwritten/removed across products.
func scopedGroupKey(owner *int64, consoleProductID *int64, baseKey string) string {
	key := baseKey
	if owner != nil && *owner > 0 {
		if consoleProductID != nil && *consoleProductID > 0 {
			return fmt.Sprintf("u:%d:p:%d:%s", *owner, *consoleProductID, key)
		}
		return fmt.Sprintf("u:%d:%s", *owner, key)
	}
	if consoleProductID != nil && *consoleProductID > 0 {
		return fmt.Sprintf("p:%d:%s", *consoleProductID, key)
	}
	return key
}

// groupKeyMatchesProductScope mirrors the repository product filter for keys that
// already carry a :p:<id>: segment. Legacy keys without a product segment return
// true so SQL can fall back to finding→run product linkage.
func groupKeyMatchesProductScope(groupKey string, ownerUserID, consoleProductID *int64) bool {
	if consoleProductID == nil || *consoleProductID <= 0 {
		return true
	}
	product := *consoleProductID
	if ownerUserID != nil && *ownerUserID > 0 {
		want := fmt.Sprintf("u:%d:p:%d:", *ownerUserID, product)
		if strings.HasPrefix(groupKey, want) {
			return true
		}
		ownerProductPrefix := fmt.Sprintf("u:%d:p:", *ownerUserID)
		if strings.HasPrefix(groupKey, ownerProductPrefix) {
			return false
		}
	}
	wantProductOnly := fmt.Sprintf("p:%d:", product)
	if strings.HasPrefix(groupKey, wantProductOnly) {
		return true
	}
	if strings.HasPrefix(groupKey, "p:") {
		return false
	}
	return true
}
