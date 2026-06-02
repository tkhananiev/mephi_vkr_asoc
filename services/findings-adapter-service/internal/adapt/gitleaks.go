package adapt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"mephi_vkr_asoc/services/findings-adapter-service/internal/models"
)

func Gitleaks(raw []byte) ([]models.FindingItem, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	var gl []models.GitleaksFinding
	if err := json.Unmarshal(raw, &gl); err != nil {
		return nil, fmt.Errorf("decode gitleaks json: %w", err)
	}
	return findingsFromGitleaks(gl), nil
}

func findingsFromGitleaks(gl []models.GitleaksFinding) []models.FindingItem {
	findings := make([]models.FindingItem, 0, len(gl))
	for _, f := range gl {
		path := strings.TrimSpace(f.File)
		if path == "" {
			path = "unknown"
		}
		id := strings.TrimSpace(f.RuleID)
		if id == "" {
			id = "gitleaks"
		}
		meta := map[string]any{
			"description": f.Description,
			"line":        f.StartLine,
		}
		if len(f.Tags) > 0 {
			meta["tags"] = f.Tags
		}
		findings = append(findings, models.FindingItem{
			AssetID:    filepath.Base(path),
			Identifier: id,
			Severity:   "high",
			Component:  path,
			Metadata:   meta,
			RawPayload: map[string]any{
				"rule_id":     f.RuleID,
				"fingerprint": f.Fingerprint,
			},
		})
	}
	return findings
}
