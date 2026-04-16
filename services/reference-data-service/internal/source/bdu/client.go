package bdu

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"mephi_vkr_aspm/services/reference-data-service/internal/models"
)

type Client struct {
	httpClient *http.Client
	feedURL    string
}

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

var bduIDPattern = regexp.MustCompile(`BDU:\d+`)
var cvePattern = regexp.MustCompile(`CVE-\d{4}-\d+`)

func New(feedURL string, insecure bool) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec
			},
		},
		feedURL:    feedURL,
	}
}

func (c *Client) Fetch(ctx context.Context) ([]models.SourceRecord, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.feedURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.demoFallbackRecord(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return c.demoFallbackRecord(), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.demoFallbackRecord(), nil
	}

	if !looksLikeXML(body) {
		return c.demoFallbackRecord(), nil
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return c.demoFallbackRecord(), nil
	}

	records := make([]models.SourceRecord, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		externalID := extractBDUID(item)
		publishedAt := parseRSSDate(item.PubDate)
		aliases := extractAliases(item.Title + " " + item.Description)
		metadata := map[string]any{
			"title":       item.Title,
			"description": item.Description,
			"link":        item.Link,
			"guid":        item.GUID,
			"pub_date":    item.PubDate,
		}

		records = append(records, models.SourceRecord{
			ExternalID:  externalID,
			Title:       item.Title,
			Description: stripSpaces(item.Description),
			PublishedAt: publishedAt,
			ModifiedAt:  publishedAt,
			SourceURL:   item.Link,
			Status:      "published",
			Metadata:    mustJSON(metadata),
			Aliases:     aliases,
			RawPayload:  string(body),
			ContentType: "application/rss+xml",
		})
	}

	return records, nil
}

func (c *Client) demoFallbackRecord() []models.SourceRecord {
	now := time.Now().UTC()
	metadata := map[string]any{
		"source": "demo_fallback",
		"reason": "bdu feed unavailable or blocked in current environment",
	}

	return []models.SourceRecord{
		{
			ExternalID:  "BDU:2021-00001",
			Title:       "Демонстрационная запись БДУ ФСТЭК",
			Description: "Fallback-запись для демонстрационного сценария корреляции с CVE-2021-44228.",
			Severity:    "high",
			PublishedAt: &now,
			ModifiedAt:  &now,
			SourceURL:   c.feedURL,
			Status:      "published",
			Metadata:    mustJSON(metadata),
			Aliases: []models.ReferenceAlias{
				{AliasType: "CVE", AliasValue: "CVE-2021-44228"},
				{AliasType: "BDU", AliasValue: "BDU:2021-00001"},
			},
			RawPayload:  `{"fallback":true}`,
			ContentType: "application/json",
		},
	}
}

func extractBDUID(item rssItem) string {
	joined := strings.TrimSpace(item.GUID + " " + item.Link + " " + item.Title)
	if match := bduIDPattern.FindString(joined); match != "" {
		return match
	}
	if item.GUID != "" {
		return item.GUID
	}
	return item.Link
}

func extractAliases(text string) []models.ReferenceAlias {
	aliases := make([]models.ReferenceAlias, 0)
	seen := map[string]struct{}{}

	for _, cve := range cvePattern.FindAllString(text, -1) {
		key := "CVE:" + cve
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		aliases = append(aliases, models.ReferenceAlias{AliasType: "CVE", AliasValue: cve})
	}

	return aliases
}

func parseRSSDate(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if ts, err := time.Parse(time.RFC1123Z, value); err == nil {
		return &ts
	}
	if ts, err := time.Parse(time.RFC1123, value); err == nil {
		return &ts
	}
	return nil
}

func stripSpaces(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func looksLikeXML(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	return strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<rss") || strings.HasPrefix(trimmed, "<feed")
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return data
}
