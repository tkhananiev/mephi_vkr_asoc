package service

import (
	"fmt"
	"strings"

	"mephi_vkr_asoc/services/jira-integration-service/internal/models"
)

func formatTicketDescription(req models.TicketRequest) string {
	var b strings.Builder
	groupRows := [][2]string{
		{"Ключ группы", req.GroupKey},
		{"Критичность (max)", fieldOrDash(req.Severity)},
		{"Затронуто активов", fmt.Sprintf("%d", req.AssetsCount)},
	}
	if ref := strings.TrimSpace(req.CorrelationRef); ref != "" && ref != strings.TrimSpace(req.GroupKey) {
		groupRows = append(groupRows, [2]string{"Корреляция", ref})
	}
	writeWikiTable(&b, "Группа", groupRows)

	if len(req.Vulnerabilities) == 0 {
		b.WriteString("\n_Детали уязвимостей не переданы._\n")
		return b.String()
	}

	for i, v := range req.Vulnerabilities {
		title := "Уязвимость"
		if len(req.Vulnerabilities) > 1 {
			title = fmt.Sprintf("Уязвимость %d", i+1)
		}
		crit := fieldOrDash(v.Criticality)
		if src := strings.TrimSpace(v.CriticalitySource); src != "" {
			crit = crit + " (" + src + ")"
		}
		writeWikiTable(&b, title, [][2]string{
			{"Файл / класс", fieldOrDash(v.AssetPath)},
			{"CVE", fieldOrDash(v.CVE)},
			{"Описание CVE", fieldOrDash(v.CVEDescription)},
			{"BDU", fieldOrDash(v.BDUID)},
			{"Описание BDU", fieldOrDash(v.BDUDescription)},
			{"Критичность", crit},
		})
	}
	return strings.TrimSpace(b.String())
}

func writeWikiTable(b *strings.Builder, section string, rows [][2]string) {
	fmt.Fprintf(b, "h3. %s\n", section)
	b.WriteString("||Поле||Значение||\n")
	for _, row := range rows {
		writeWikiTableRow(b, row[0], row[1])
	}
	b.WriteString("\n")
}

func writeWikiTableRow(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "|%s|%s|\n", wikiCell(label), wikiCell(value))
}

func wikiCell(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	if s == "" {
		return "—"
	}
	return s
}

func fieldOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return strings.TrimSpace(s)
}

const jiraSummaryMaxLen = 255

func ticketSummary(req models.TicketRequest) string {
	if len(req.Vulnerabilities) > 0 {
		v := req.Vulnerabilities[0]
		filePart := ""
		if p := strings.TrimSpace(v.AssetPath); p != "" {
			filePart = filepathBase(p)
		}
		var body string
		if d := strings.TrimSpace(v.CVEDescription); d != "" {
			body = joinSummaryPrefix(strings.TrimSpace(v.CVE), d)
		} else if d := strings.TrimSpace(v.BDUDescription); d != "" {
			body = joinSummaryPrefix(strings.TrimSpace(v.BDUID), d)
		} else if cve := strings.TrimSpace(v.CVE); cve != "" {
			body = cve
		} else if bdu := strings.TrimSpace(v.BDUID); bdu != "" {
			body = bdu
		}
		switch {
		case filePart != "" && body != "":
			return truncateJiraSummary(filePart + " · " + body)
		case filePart != "":
			return truncateJiraSummary(filePart)
		case body != "":
			return truncateJiraSummary(body)
		}
	}
	return truncateJiraSummary(req.GroupKey)
}

func filepathBase(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func joinSummaryPrefix(id, description string) string {
	if id == "" {
		return description
	}
	return id + ": " + description
}

func truncateJiraSummary(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= jiraSummaryMaxLen {
		return s
	}
	return s[:jiraSummaryMaxLen-1] + "…"
}
