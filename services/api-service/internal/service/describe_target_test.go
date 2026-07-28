package service

import (
	"testing"

	"mephi_vkr_asoc/services/api-service/internal/models"
)

func TestDescribeScanTargetRedactsUserinfo(t *testing.T) {
	got := DescribeScanTarget(models.ScanRequest{
		GitRepositoryURL: "https://ghp_LEAKME_TOKEN@github.com/org/repo.git",
		GitRepositoryRef: "main",
	})
	if got != "https://github.com/org/repo.git@main" {
		t.Fatalf("got %q", got)
	}
}
