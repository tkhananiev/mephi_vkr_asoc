package adapt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"mephi_vkr_asoc/services/findings-adapter-service/internal/models"
)

type trivyReport struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target          string      `json:"Target"`
	Vulnerabilities []trivyVuln `json:"Vulnerabilities"`
}

type trivyVuln struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title"`
	Description      string `json:"Description"`
	PrimaryURL       string `json:"PrimaryURL"`
}

func Trivy(raw []byte) ([]models.FindingItem, error) {
	rep, err := parseTrivyReport(raw)
	if err != nil {
		return nil, fmt.Errorf("decode trivy json: %w", err)
	}
	return findingsFromTrivyReport(rep), nil
}

func parseTrivyReport(raw []byte) (trivyReport, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return trivyReport{}, nil
	}
	var rep trivyReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return trivyReport{}, err
	}
	return rep, nil
}

func findingsFromTrivyReport(rep trivyReport) []models.FindingItem {
	findings := make([]models.FindingItem, 0, 32)
	for _, res := range rep.Results {
		target := strings.TrimSpace(res.Target)
		if target == "" {
			target = "unknown"
		}
		for _, v := range res.Vulnerabilities {
			id := strings.TrimSpace(v.VulnerabilityID)
			if id == "" {
				id = "trivy-vuln"
			}
			pkg := strings.TrimSpace(v.PkgName)
			if pkg == "" {
				pkg = target
			}
			meta := map[string]any{
				"title":       v.Title,
				"description": v.Description,
				"pkg_name":    v.PkgName,
			}
			if v.PrimaryURL != "" {
				meta["primary_url"] = v.PrimaryURL
			}
			findings = append(findings, models.FindingItem{
				AssetID:    pkg,
				Identifier: id,
				Severity:   trivySeverity(v.Severity),
				Component:  target,
				Version:    strings.TrimSpace(v.InstalledVersion),
				CVE:        id,
				Metadata:   meta,
				RawPayload: map[string]any{
					"vulnerability_id": id,
					"fixed_version":    v.FixedVersion,
				},
			})
		}
	}
	return findings
}

func trivySeverity(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL":
		return "critical"
	case "HIGH":
		return "high"
	case "MEDIUM":
		return "medium"
	case "LOW":
		return "low"
	case "UNKNOWN":
		return "unknown"
	default:
		return NormalizeSeverity(strings.ToLower(s))
	}
}
