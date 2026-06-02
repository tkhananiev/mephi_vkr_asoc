package service

import (
	"strings"
	"testing"

	"mephi_vkr_asoc/services/jira-integration-service/internal/models"
)

func TestTicketSummary_usesCVEDescription(t *testing.T) {
	got := ticketSummary(models.TicketRequest{
		GroupKey: "u:1:cve:CVE-2021-44228",
		Vulnerabilities: []models.TicketVulnerabilityDetail{
			{
				AssetPath:      "src/main/java/App.java",
				CVE:            "CVE-2021-44228",
				CVEDescription: "Log4Shell remote code execution",
			},
		},
	})
	want := "App.java · CVE-2021-44228: Log4Shell remote code execution"
	if got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}

func TestFormatTicketDescription_wikiTable(t *testing.T) {
	desc := formatTicketDescription(models.TicketRequest{
		GroupKey:    "u:1:cve:CVE-2021-44228",
		Severity:    "high",
		AssetsCount: 2,
		Vulnerabilities: []models.TicketVulnerabilityDetail{
			{
				AssetPath:         "src/main/java/App.java",
				CVE:               "CVE-2021-44228",
				CVEDescription:    "Log4Shell demo",
				BDUID:             "BDU:2021-00001",
				BDUDescription:    "BDU demo text",
				Criticality:       "critical",
				CriticalitySource: "CVSS (NVD)",
			},
		},
	})
	for _, want := range []string{
		"||Поле||Значение||",
		"|Файл / класс|src/main/java/App.java|",
		"|Описание CVE|Log4Shell demo|",
		"|Критичность|critical (CVSS (NVD))|",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("description missing %q:\n%s", want, desc)
		}
	}
}
