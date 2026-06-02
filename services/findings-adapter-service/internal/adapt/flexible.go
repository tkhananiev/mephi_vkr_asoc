package adapt

import (
	"bytes"
	"encoding/json"
	"fmt"

	"mephi_vkr_asoc/services/findings-adapter-service/internal/models"
)

// Flexible определяет формат по первому ключу JSON (semgrep / trivy / findings / gitleaks array).
func Flexible(raw []byte, targetURL string) ([]models.FindingItem, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty response body")
	}
	switch raw[0] {
	case '[':
		var norm []models.FindingItem
		if err := json.Unmarshal(raw, &norm); err == nil {
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
		if _, ok := probe["findings"]; ok {
			var wrap struct {
				Findings []models.FindingItem `json:"findings"`
			}
			if err := json.Unmarshal(raw, &wrap); err != nil {
				return nil, err
			}
			if wrap.Findings == nil {
				return []models.FindingItem{}, nil
			}
			return wrap.Findings, nil
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
