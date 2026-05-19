package service

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"mephi_vkr_asoc/services/api-service/internal/models"
)

var cveIDPattern = regexp.MustCompile(`(?i)CVE-\d{4}-\d+`)
var cweIDPattern = regexp.MustCompile(`(?i)CWE-\d+`)

func semgrepCWEFromMetadata(meta json.RawMessage) string {
	s := semgrepFlexibleString(meta, "cwe")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if m := cweIDPattern.FindString(s); m != "" {
		return strings.ToUpper(m)
	}
	digits := true
	for _, r := range s {
		if r < '0' || r > '9' {
			digits = false
			break
		}
	}
	if digits && len(s) > 0 {
		return "CWE-" + strings.ToUpper(s)
	}
	return ""
}

func semgrepCVEFromMetadata(meta json.RawMessage) string {
	s := strings.TrimSpace(semgrepFlexibleString(meta, "cve"))
	if s != "" {
		if x := strings.ToUpper(cveIDPattern.FindString(s)); x != "" {
			return x
		}
	}
	refBlob := metaValue(meta, "references")
	if len(refBlob) == 0 {
		return ""
	}
	var refs []string
	if err := json.Unmarshal(refBlob, &refs); err != nil || len(refs) == 0 {
		var single string
		if err := json.Unmarshal(refBlob, &single); err == nil && strings.TrimSpace(single) != "" {
			refs = []string{single}
		}
	}
	for _, r := range refs {
		if m := cveIDPattern.FindString(r); m != "" {
			return strings.ToUpper(m)
		}
	}
	return ""
}

func semgrepFlexibleString(meta json.RawMessage, key string) string {
	raw := metaValue(meta, key)
	if len(raw) == 0 {
		return ""
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, x := range arr {
			x = strings.TrimSpace(x)
			if x != "" {
				return x
			}
		}
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return ""
}

func metaValue(meta json.RawMessage, key string) json.RawMessage {
	if len(meta) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(meta, &m); err != nil {
		return nil
	}
	return m[key]
}

func findingsFromSemgrepResult(sr models.SemgrepResult) []models.ProcessingFindingItem {
	findings := make([]models.ProcessingFindingItem, 0, len(sr.Results))
	for _, result := range sr.Results {
		meta := result.Extra.Metadata
		cwe := semgrepCWEFromMetadata(meta)
		cve := semgrepCVEFromMetadata(meta)

		findings = append(findings, models.ProcessingFindingItem{
			AssetID:    filepath.Base(result.Path),
			Identifier: result.CheckID,
			Severity:   normalizeSeverity(result.Extra.Severity),
			Component:  result.Path,
			Version:    "",
			CVE:        strings.TrimSpace(cve),
			CWE:        cwe,
			Metadata: map[string]any{
				"message": result.Extra.Message,
				"path":    result.Path,
			},
			RawPayload: map[string]any{
				"check_id": result.CheckID,
			},
		})
	}
	return findings
}
