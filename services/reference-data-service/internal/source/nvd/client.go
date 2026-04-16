package nvd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mephi_vkr_aspm/services/reference-data-service/internal/models"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type apiResponse struct {
	Vulnerabilities []struct {
		CVE struct {
			ID               string `json:"id"`
			Published        string `json:"published"`
			LastModified     string `json:"lastModified"`
			VulnStatus       string `json:"vulnStatus"`
			Descriptions     []struct {
				Lang  string `json:"lang"`
				Value string `json:"value"`
			} `json:"descriptions"`
			Weaknesses []struct {
				Description []struct {
					Value string `json:"value"`
				} `json:"description"`
			} `json:"weaknesses"`
			Metrics map[string][]struct {
				CVSSData struct {
					BaseScore float64 `json:"baseScore"`
				} `json:"cvssData"`
			} `json:"metrics"`
		} `json:"cve"`
	} `json:"vulnerabilities"`
}

func New(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 45 * time.Second},
		baseURL:    baseURL,
	}
}

func (c *Client) Fetch(ctx context.Context) ([]models.SourceRecord, error) {
	return c.fetch(ctx, "")
}

func (c *Client) FetchByCVE(ctx context.Context, cveID string) ([]models.SourceRecord, error) {
	return c.fetch(ctx, cveID)
}

func (c *Client) fetch(ctx context.Context, cveID string) ([]models.SourceRecord, error) {
	url := c.baseURL + "?resultsPerPage=20"
	if strings.TrimSpace(cveID) != "" {
		url = c.baseURL + "?cveId=" + strings.TrimSpace(cveID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nvd api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload apiResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	records := make([]models.SourceRecord, 0, len(payload.Vulnerabilities))
	for _, entry := range payload.Vulnerabilities {
		cve := entry.CVE
		publishedAt := parseTime(cve.Published)
		modifiedAt := parseTime(cve.LastModified)
		description := firstDescription(cve.Descriptions)
		severity := firstSeverity(cve.Metrics)
		metadata := map[string]any{
			"cve_id":        cve.ID,
			"published":     cve.Published,
			"last_modified": cve.LastModified,
			"status":        cve.VulnStatus,
			"weaknesses":    cve.Weaknesses,
		}

		records = append(records, models.SourceRecord{
			ExternalID:  cve.ID,
			Title:       cve.ID,
			Description: description,
			Severity:    severity,
			PublishedAt: publishedAt,
			ModifiedAt:  modifiedAt,
			SourceURL:   "https://nvd.nist.gov/vuln/detail/" + cve.ID,
			Status:      cve.VulnStatus,
			Metadata:    mustJSON(metadata),
			Aliases: []models.ReferenceAlias{
				{AliasType: "CVE", AliasValue: cve.ID},
			},
			RawPayload:  string(body),
			ContentType: "application/json",
		})
	}

	return records, nil
}

func firstDescription(items []struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}) string {
	for _, item := range items {
		if strings.EqualFold(item.Lang, "en") {
			return item.Value
		}
	}
	if len(items) > 0 {
		return items[0].Value
	}
	return ""
}

func firstSeverity(metrics map[string][]struct {
	CVSSData struct {
		BaseScore float64 `json:"baseScore"`
	} `json:"cvssData"`
}) string {
	for _, versions := range metrics {
		if len(versions) > 0 {
			return fmt.Sprintf("%.1f", versions[0].CVSSData.BaseScore)
		}
	}
	return ""
}

func parseTime(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return &ts
	}
	return nil
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return data
}
