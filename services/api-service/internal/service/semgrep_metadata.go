package service

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	cwePattern = regexp.MustCompile(`(?i)\bcwe[-_ ]?(\d+)\b`)
	cvePattern = regexp.MustCompile(`(?i)\bcve-\d{4}-\d{4,}\b`)
)

func semgrepCWEFromMetadata(meta json.RawMessage) string {
	var obj map[string]any
	if err := json.Unmarshal(meta, &obj); err != nil {
		return ""
	}
	return firstNormalizedCWE(obj["cwe"])
}

func semgrepCVEFromMetadata(meta json.RawMessage) string {
	var obj map[string]any
	if err := json.Unmarshal(meta, &obj); err != nil {
		return ""
	}
	if cve := firstNormalizedCVE(obj["cve"]); cve != "" {
		return cve
	}
	return firstNormalizedCVE(obj["references"])
}

func firstNormalizedCWE(value any) string {
	switch v := value.(type) {
	case string:
		return normalizeCWE(v)
	case []any:
		for _, item := range v {
			if cwe := firstNormalizedCWE(item); cwe != "" {
				return cwe
			}
		}
	}
	return ""
}

func normalizeCWE(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if allDigits(value) {
		return "CWE-" + value
	}
	matches := cwePattern.FindStringSubmatch(value)
	if len(matches) == 2 {
		return "CWE-" + matches[1]
	}
	return ""
}

func firstNormalizedCVE(value any) string {
	switch v := value.(type) {
	case string:
		return normalizeCVE(v)
	case []any:
		for _, item := range v {
			if cve := firstNormalizedCVE(item); cve != "" {
				return cve
			}
		}
	}
	return ""
}

func normalizeCVE(value string) string {
	match := cvePattern.FindString(value)
	if match == "" {
		return ""
	}
	return strings.ToUpper(match)
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
