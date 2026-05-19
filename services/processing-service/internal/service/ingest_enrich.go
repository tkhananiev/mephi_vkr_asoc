package service

import (
	"fmt"
	"regexp"
	"strings"

	"mephi_vkr_asoc/services/processing-service/internal/models"
)

var ingestCVEPat = regexp.MustCompile(`(?i)CVE-\d{4}-\d+`)
var ingestCWEPat = regexp.MustCompile(`(?i)CWE-\d+`)

// enrichFindingCatalogFields — CWE/CVE могут оказаться только в metadata (CI, старый формат).
func enrichFindingCatalogFields(item models.FindingDTO) models.FindingDTO {
	item.CVE = strings.TrimSpace(item.CVE)
	item.CWE = strings.TrimSpace(item.CWE)
	if item.Metadata == nil {
		return item
	}
	if item.CWE == "" {
		if cw := flexibleMetaString(item.Metadata, "cwe"); cw != "" {
			if m := ingestCWEPat.FindString(cw); m != "" {
				item.CWE = strings.ToUpper(m)
			}
		}
	}
	if item.CVE == "" {
		if cv := flexibleMetaString(item.Metadata, "cve"); cv != "" {
			if m := ingestCVEPat.FindString(cv); m != "" {
				item.CVE = strings.ToUpper(m)
			}
		}
		if item.CVE == "" {
			if refs, ok := item.Metadata["references"]; ok {
				item.CVE = firstCVEInAny(refs)
			}
		}
	}
	return item
}

func flexibleMetaString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []any:
		for _, el := range x {
			if s := flexibleMetaScalarString(el); s != "" {
				return s
			}
		}
	case []string:
		for _, s := range x {
			if ts := strings.TrimSpace(s); ts != "" {
				return ts
			}
		}
	}
	return ""
}

func flexibleMetaScalarString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func firstCVEInAny(v any) string {
	switch x := v.(type) {
	case string:
		if m := ingestCVEPat.FindString(x); m != "" {
			return strings.ToUpper(m)
		}
	case []any:
		for _, el := range x {
			if s := firstCVEInAny(el); s != "" {
				return s
			}
		}
	case []string:
		for _, el := range x {
			if m := ingestCVEPat.FindString(el); m != "" {
				return strings.ToUpper(m)
			}
		}
	}
	return ""
}
