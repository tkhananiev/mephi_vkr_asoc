package service

import (
	"context"
	"testing"

	"mephi_vkr_asoc/services/processing-service/internal/models"
)

type stubProcessingRepo struct {
	cweRefID   int64
	cweCVEAlias string

	created []models.Vulnerability
	groups  []string
}

func (s *stubProcessingRepo) PurgeScannerScope(context.Context, string, *int64, *int64) error {
	return nil
}
func (s *stubProcessingRepo) StartRun(context.Context, string, int, *int64, string, *int64) (int64, error) {
	return 1, nil
}
func (s *stubProcessingRepo) FinishRun(context.Context, int64, string, models.ProcessingResult, *string) error {
	return nil
}
func (s *stubProcessingRepo) InsertFinding(context.Context, models.Finding) (int64, error) {
	return 10, nil
}
func (s *stubProcessingRepo) FindReferenceRecordIDByCVE(context.Context, string) (*int64, error) {
	return nil, nil
}
func (s *stubProcessingRepo) FindReferenceRecordIDByCWE(_ context.Context, cwe string) (*int64, error) {
	if cwe == "" {
		return nil, nil
	}
	id := s.cweRefID
	return &id, nil
}
func (s *stubProcessingRepo) FindCVEAliasByReferenceRecordID(context.Context, int64) (string, error) {
	return s.cweCVEAlias, nil
}
func (s *stubProcessingRepo) CreateVulnerability(_ context.Context, vulnerability models.Vulnerability) (int64, bool, error) {
	s.created = append(s.created, vulnerability)
	return int64(len(s.created)), true, nil
}
func (s *stubProcessingRepo) MergeVulnerabilityCatalog(context.Context, int64, string, *int64, string) error {
	return nil
}
func (s *stubProcessingRepo) LinkFindingToVulnerability(context.Context, int64, int64) error {
	return nil
}
func (s *stubProcessingRepo) UpsertGroup(_ context.Context, groupKey, _, _ string) (int64, bool, error) {
	s.groups = append(s.groups, groupKey)
	return 100, true, nil
}
func (s *stubProcessingRepo) LinkGroupToVulnerability(context.Context, int64, int64) error {
	return nil
}
func (s *stubProcessingRepo) ListGroups(context.Context, int, *int64, *int64, string) ([]models.VulnerabilityGroup, error) {
	return nil, nil
}
func (s *stubProcessingRepo) UpdateGroupStatus(context.Context, int64, string, *int64) (models.VulnerabilityGroup, error) {
	return models.VulnerabilityGroup{}, nil
}
func (s *stubProcessingRepo) ListVulnerabilityReport(context.Context, int, *int64, *int64, *models.VulnerabilityReportFilter) ([]models.VulnerabilityReportRow, error) {
	return nil, nil
}
func (s *stubProcessingRepo) GetGroupJiraContext(context.Context, int64, *int64, *int64) (models.GroupJiraContext, error) {
	return models.GroupJiraContext{}, nil
}

func TestProcessFindings_CWEMatchDoesNotInventCVE(t *testing.T) {
	repo := &stubProcessingRepo{
		cweRefID:    42,
		cweCVEAlias: "CVE-2024-99999",
	}
	svc := New(repo)

	_, err := svc.ProcessFindings(context.Background(), models.IngestRequest{
		ScannerName: "semgrep",
		Findings: []models.FindingDTO{{
			AssetID:    "Login.java",
			Identifier: "java.lang.security.audit.sqli",
			Severity:   "high",
			Component:  "src/Login.java",
			CWE:        "CWE-89",
		}},
	})
	if err != nil {
		t.Fatalf("ProcessFindings: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("want 1 vulnerability, got %d", len(repo.created))
	}
	got := repo.created[0]
	if got.CVEID != "" {
		t.Fatalf("CWE-only finding invented CVE %q", got.CVEID)
	}
	if got.CorrelationStatus != "matched_by_cwe" {
		t.Fatalf("correlation status: got %q", got.CorrelationStatus)
	}
	if got.ReferenceRecordID == nil || *got.ReferenceRecordID != 42 {
		t.Fatalf("reference record: %+v", got.ReferenceRecordID)
	}
	if len(repo.groups) != 1 {
		t.Fatalf("want 1 group, got %d", len(repo.groups))
	}
	if repo.groups[0] != "::CWE-89::src/Login.java::" {
		t.Fatalf("group key: %q", repo.groups[0])
	}
}
