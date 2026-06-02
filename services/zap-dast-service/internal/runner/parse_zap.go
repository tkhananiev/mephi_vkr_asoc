package runner

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type zapJSONReport struct {
	Site []zapSite `json:"site"`
}

type zapSite struct {
	Name   string     `json:"@name"`
	Alerts []zapAlert `json:"alerts"`
}

type zapAlert struct {
	PluginID   string `json:"pluginid"`
	AlertRef   string `json:"alertRef"`
	Alert      string `json:"alert"`
	Name       string `json:"name"`
	RiskCode   string `json:"riskcode"`
	Confidence string `json:"confidence"`
	RiskDesc   string `json:"riskdesc"`
	Desc       string `json:"desc"`
	URI        string `json:"uri"`
	Param      string `json:"param"`
	CWEID      string `json:"cweid"`
	WASCID     string `json:"wascid"`
	Solution   string `json:"solution"`
}

func findingsFromZAPReport(raw []byte, targetURL string) ([]normalizedFinding, error) {
	raw = bytesTrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty ZAP JSON report")
	}
	var rep zapJSONReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, fmt.Errorf("decode ZAP JSON: %w", err)
	}

	host := hostFromTarget(targetURL)
	out := make([]normalizedFinding, 0, 32)
	seen := make(map[string]struct{})

	for _, site := range rep.Site {
		for _, a := range site.Alerts {
			id := zapAlertIdentifier(a)
			if id == "" {
				continue
			}
			uri := strings.TrimSpace(a.URI)
			dedupeKey := id + "|" + uri
			if _, dup := seen[dedupeKey]; dup {
				continue
			}
			seen[dedupeKey] = struct{}{}

			title := strings.TrimSpace(a.Name)
			if title == "" {
				title = strings.TrimSpace(a.Alert)
			}
			meta := map[string]any{
				"title":   title,
				"engine":  "owasp-zap",
				"plugin":  strings.TrimSpace(a.PluginID),
				"uri":     uri,
				"message": strings.TrimSpace(a.Desc),
			}
			if c := strings.TrimSpace(a.Confidence); c != "" {
				meta["confidence"] = c
			}
			if rd := strings.TrimSpace(a.RiskDesc); rd != "" {
				meta["riskdesc"] = rd
			}
			if p := strings.TrimSpace(a.Param); p != "" {
				meta["param"] = p
			}

			cwe := strings.TrimSpace(a.CWEID)
			if cwe != "" && !strings.HasPrefix(strings.ToUpper(cwe), "CWE-") {
				cwe = "CWE-" + cwe
			}

			out = append(out, normalizedFinding{
				AssetID:    host,
				Identifier: id,
				Severity:   zapRiskToSeverity(a.RiskCode),
				Component:  firstNonEmpty(uri, targetURL),
				CWE:        cwe,
				Metadata:   meta,
				RawPayload: map[string]any{
					"pluginid": a.PluginID,
					"alert":    a.Alert,
					"uri":      uri,
					"wascid":   strings.TrimSpace(a.WASCID),
				},
			})
		}
	}
	return out, nil
}

func zapAlertIdentifier(a zapAlert) string {
	if p := strings.TrimSpace(a.PluginID); p != "" {
		return "zap-" + p
	}
	if r := strings.TrimSpace(a.AlertRef); r != "" {
		return "zap-" + r
	}
	name := strings.TrimSpace(a.Name)
	if name == "" {
		name = strings.TrimSpace(a.Alert)
	}
	if name == "" {
		return ""
	}
	return "zap-" + slugIdentifier(name)
}

func zapRiskToSeverity(riskCode string) string {
	switch strings.TrimSpace(riskCode) {
	case "3":
		return "high"
	case "2":
		return "medium"
	case "1":
		return "low"
	case "0":
		return "info"
	default:
		if n, err := strconv.Atoi(strings.TrimSpace(riskCode)); err == nil {
			switch {
			case n >= 3:
				return "high"
			case n == 2:
				return "medium"
			case n == 1:
				return "low"
			default:
				return "info"
			}
		}
		rd := strings.ToLower(strings.TrimSpace(riskCode))
		if strings.Contains(rd, "high") {
			return "high"
		}
		if strings.Contains(rd, "medium") {
			return "medium"
		}
		if strings.Contains(rd, "low") {
			return "low"
		}
		return "info"
	}
}

func hostFromTarget(targetURL string) string {
	u, err := validateTargetURL(targetURL)
	if err != nil {
		return "target"
	}
	return u.Hostname()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func slugIdentifier(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
