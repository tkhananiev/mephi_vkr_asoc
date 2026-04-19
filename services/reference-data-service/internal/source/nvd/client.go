package nvd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mephi_vkr_aspm/services/reference-data-service/internal/models"
)

const (
	maxResultsPerPageNVD = 2000
	// NVD: без ключа ~5 запросов / 30 с; с ключом ~50 / 30 с.
	throttleNoAPIKey  = 6 * time.Second
	throttleWithAPIKey = 650 * time.Millisecond
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	pageSize   int
	maxPages   int // 0 = без ограничения (все страницы)
}

type apiResponse struct {
	TotalResults    int `json:"totalResults"`
	ResultsPerPage  int `json:"resultsPerPage"`
	StartIndex      int `json:"startIndex"`
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

// New создаёт клиент NVD API 2.0. apiKey — опционально (заголовок apiKey, выше лимит запросов).
// pageSize — до 2000; maxPages — ограничение числа страниц за один вызов Fetch (0 = все).
func New(baseURL, apiKey string, pageSize, maxPages int) *Client {
	if pageSize <= 0 || pageSize > maxResultsPerPageNVD {
		pageSize = maxResultsPerPageNVD
	}
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "?&"),
		apiKey:     strings.TrimSpace(apiKey),
		pageSize:   pageSize,
		maxPages:   maxPages,
	}
}

func (c *Client) Fetch(ctx context.Context) ([]models.SourceRecord, error) {
	return c.fetch(ctx, "")
}

func (c *Client) FetchByCVE(ctx context.Context, cveID string) ([]models.SourceRecord, error) {
	return c.fetch(ctx, cveID)
}

func (c *Client) fetch(ctx context.Context, cveID string) ([]models.SourceRecord, error) {
	if strings.TrimSpace(cveID) != "" {
		return c.fetchSingleCVE(ctx, strings.TrimSpace(cveID))
	}
	return c.fetchAllPages(ctx)
}

func (c *Client) fetchSingleCVE(ctx context.Context, cveID string) ([]models.SourceRecord, error) {
	url := fmt.Sprintf("%s?cveId=%s", c.baseURL, strings.TrimSpace(cveID))
	payload, err := c.doGET(ctx, url)
	if err != nil {
		return nil, err
	}
	return c.recordsFromPayload(payload)
}

// SyncAllPages загружает все страницы NVD без накопления всего каталога в памяти (onPage вызывается на каждую страницу).
func (c *Client) SyncAllPages(ctx context.Context, onPage func([]models.SourceRecord) error) error {
	startIndex := 0
	pageNum := 0

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		url := fmt.Sprintf("%s?resultsPerPage=%d&startIndex=%d", c.baseURL, c.pageSize, startIndex)
		payload, err := c.doGET(ctx, url)
		if err != nil {
			return err
		}

		chunk, err := c.recordsFromPayload(payload)
		if err != nil {
			return err
		}

		total := payload.TotalResults

		if len(chunk) > 0 {
			if err := onPage(chunk); err != nil {
				return err
			}
		}

		if len(chunk) == 0 {
			break
		}

		nextStart := startIndex + len(chunk)
		var done bool
		switch {
		case total > 0:
			done = nextStart >= total
		case payload.ResultsPerPage > 0:
			// При totalResults=0 в JSON не сравниваем nextStart >= total (иначе 20 >= 0 → ложный конец после первой страницы).
			done = len(chunk) < payload.ResultsPerPage
		default:
			// Нет ни total, ни resultsPerPage — идём, пока не придёт пустой chunk.
			done = false
		}

		if done {
			break
		}

		startIndex = nextStart
		pageNum++

		if c.maxPages > 0 && pageNum >= c.maxPages {
			break
		}

		if err := c.throttle(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) fetchAllPages(ctx context.Context) ([]models.SourceRecord, error) {
	var all []models.SourceRecord
	err := c.SyncAllPages(ctx, func(chunk []models.SourceRecord) error {
		all = append(all, chunk...)
		return nil
	})
	return all, err
}

func (c *Client) doGET(ctx context.Context, url string) (*apiResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("apiKey", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("nvd api returned status %d: %s", resp.StatusCode, string(bytes.TrimSpace(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload apiResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (c *Client) recordsFromPayload(payload *apiResponse) ([]models.SourceRecord, error) {
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

		rawPayload, err := json.Marshal(entry)
		if err != nil {
			rawPayload = []byte("{}")
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
			RawPayload:  string(rawPayload),
			ContentType: "application/json",
		})
	}
	return records, nil
}

func (c *Client) throttle(ctx context.Context) error {
	d := throttleNoAPIKey
	if c.apiKey != "" {
		d = throttleWithAPIKey
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
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
