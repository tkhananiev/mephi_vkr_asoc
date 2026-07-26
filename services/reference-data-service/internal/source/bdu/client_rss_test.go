package bdu

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestExtractBDUIDFullIdentifier(t *testing.T) {
	got := extractBDUID(rssItem{
		GUID:  "https://bdu.fstec.ru/vul/BDU:2024-00001",
		Title: "BDU:2024-00001 Some title",
	})
	if got != "BDU:2024-00001" {
		t.Fatalf("got %q want BDU:2024-00001", got)
	}
}

func TestExtractBDUIDDoesNotCollapseYear(t *testing.T) {
	a := extractBDUID(rssItem{Title: "BDU:2024-00001 Buffer overflow"})
	b := extractBDUID(rssItem{Title: "BDU:2024-00002 SQL injection"})
	if a == b {
		t.Fatalf("distinct BDU records collapsed to %q", a)
	}
	if a != "BDU:2024-00001" || b != "BDU:2024-00002" {
		t.Fatalf("got %q and %q", a, b)
	}
}

func TestExtractBDUIDOldPatternWouldTruncate(t *testing.T) {
	// Guard against regressing to BDU:\d+, which matched only the year prefix.
	old := regexp.MustCompile(`BDU:\d+`)
	raw := "BDU:2024-00001"
	if old.FindString(raw) != "BDU:2024" {
		t.Fatalf("test assumption failed: old pattern got %q", old.FindString(raw))
	}
	if extractBDUID(rssItem{Title: raw}) != "BDU:2024-00001" {
		t.Fatalf("new pattern must keep full id, got %q", extractBDUID(rssItem{Title: raw}))
	}
}

func TestFetchReturnsErrorWhenFeedUnavailable(t *testing.T) {
	client, err := NewHTTPClient(false, "", 2*time.Second)
	if err != nil {
		t.Fatalf("http client: %v", err)
	}
	c := &Client{
		httpClient: client,
		feedURL:    "http://127.0.0.1:1/missing-bdu-feed",
	}
	recs, err := c.Fetch(context.Background())
	if err == nil {
		t.Fatalf("expected error when RSS feed is unavailable, got %d records", len(recs))
	}
	if !strings.Contains(err.Error(), "bdu rss") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("expected no fallback records, got %#v", recs)
	}
}

func TestFetchReturnsErrorWhenFeedURLEmpty(t *testing.T) {
	c := &Client{httpClient: &http.Client{Timeout: time.Second}, feedURL: ""}
	_, err := c.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error for empty feed URL")
	}
}
