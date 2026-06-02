package httpapi

import (
	"html"
	"strings"
)

// jiraWikiToHTML — упрощённый рендер wiki-таблиц и заголовков h3. для консоли mock.
func jiraWikiToHTML(desc string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="jira-desc">`)
	lines := strings.Split(desc, "\n")
	inTable := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if inTable {
				b.WriteString("</tbody></table>")
				inTable = false
			}
			continue
		}
		if strings.HasPrefix(line, "h3. ") {
			if inTable {
				b.WriteString("</tbody></table>")
				inTable = false
			}
			b.WriteString("<h3>")
			b.WriteString(html.EscapeString(strings.TrimPrefix(line, "h3. ")))
			b.WriteString("</h3>")
			continue
		}
		if strings.HasPrefix(line, "||") && strings.HasSuffix(line, "||") {
			if inTable {
				b.WriteString("</tbody></table>")
			}
			cells := splitWikiTableRow(line)
			b.WriteString(`<table class="jira-fields"><thead><tr>`)
			for _, c := range cells {
				b.WriteString("<th>")
				b.WriteString(html.EscapeString(unescapeWikiCell(c)))
				b.WriteString("</th>")
			}
			b.WriteString("</tr></thead><tbody>")
			inTable = true
			continue
		}
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") && inTable {
			cells := splitWikiTableRow(line)
			b.WriteString("<tr>")
			for _, c := range cells {
				b.WriteString("<td>")
				b.WriteString(html.EscapeString(unescapeWikiCell(c)))
				b.WriteString("</td>")
			}
			b.WriteString("</tr>")
			continue
		}
		if inTable {
			b.WriteString("</tbody></table>")
			inTable = false
		}
		if strings.HasPrefix(line, "_") && strings.HasSuffix(line, "_") {
			b.WriteString("<p><em>")
			b.WriteString(html.EscapeString(strings.Trim(line, "_")))
			b.WriteString("</em></p>")
			continue
		}
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(line))
		b.WriteString("</p>")
	}
	if inTable {
		b.WriteString("</tbody></table>")
	}
	b.WriteString("</div>")
	return b.String()
}

func splitWikiTableRow(line string) []string {
	line = strings.Trim(line, "|")
	if line == "" {
		return nil
	}
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func unescapeWikiCell(s string) string {
	return strings.ReplaceAll(s, `\|`, "|")
}
