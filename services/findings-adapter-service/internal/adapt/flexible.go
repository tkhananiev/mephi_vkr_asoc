package adapt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"mephi_vkr_asoc/services/findings-adapter-service/internal/models"
)

func Flexible(raw []byte, targetURL string) ([]models.FindingItem, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty response body")
	}
	switch raw[0] {
	case '[':
		var norm []models.FindingItem
		if err := json.Unmarshal(raw, &norm); err == nil && looksLikeNormalizedFindings(norm) {
			return norm, nil
		}
		return Gitleaks(raw)
	case '{':
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, err
		}
		if _, ok := probe["results"]; ok {
			return Semgrep(raw)
		}
		if inner, ok := probe["findings"]; ok {
			var wrap struct {
				Findings []models.FindingItem `json:"findings"`
			}
			if err := json.Unmarshal(raw, &wrap); err != nil {
				return nil, err
			}
			if wrap.Findings == nil {
				return []models.FindingItem{}, nil
			}
			if looksLikeNormalizedFindings(wrap.Findings) {
				return wrap.Findings, nil
			}
			// Native scanner arrays (e.g. Gitleaks) under "findings" decode as blank
			// FindingItems because unknown fields are ignored — fall through.
			return Gitleaks(inner)
		}
		if _, ok := probe["Results"]; ok {
			return Trivy(raw)
		}
		if _, ok := probe["site"]; ok {
			return ZAP(raw, targetURL)
		}
		return nil, fmt.Errorf(`JSON object must contain "results" (semgrep), "findings", "Results" (trivy) or "site" (zap)`)
	default:
		return nil, fmt.Errorf("response must be JSON array or object")
	}
}

// looksLikeNormalizedFindings reports whether a decoded slice is already in
// ASOC FindingItem shape. Scanner-native objects also decode into FindingItem
// without error (unknown JSON fields ignored), leaving blank identifiers.
func looksLikeNormalizedFindings(items []models.FindingItem) bool {
	if len(items) == 0 {
		return true
	}
	for _, it := range items {
		if strings.TrimSpace(it.AssetID) != "" ||
			strings.TrimSpace(it.Identifier) != "" ||
			strings.TrimSpace(it.Severity) != "" ||
			strings.TrimSpace(it.Component) != "" ||
			strings.TrimSpace(it.Version) != "" ||
			strings.TrimSpace(it.CVE) != "" ||
			strings.TrimSpace(it.CWE) != "" {
			return true
		}
	}
	return false
}
