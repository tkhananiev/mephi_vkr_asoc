package httpapi

import (
	"path/filepath"
	"strings"
)

func wikiFieldValue(desc, label string) string {
	for _, line := range strings.Split(desc, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "||") {
			continue
		}
		cells := splitWikiTableRow(line)
		if len(cells) >= 2 && cells[0] == label {
			v := unescapeWikiCell(cells[1])
			if v != "" && v != "—" {
				return v
			}
		}
	}
	return ""
}

func displayAssetPath(fullPath string) string {
	fullPath = strings.TrimSpace(fullPath)
	if fullPath == "" || fullPath == "—" {
		return "—"
	}
	base := filepath.Base(fullPath)
	if base != "" && base != "." && base != "/" {
		return base + " (" + fullPath + ")"
	}
	return fullPath
}
