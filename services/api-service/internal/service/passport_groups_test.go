package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mephi_vkr_asoc/services/api-service/internal/models"
)

func TestPassportAfterFindingsSkipsGroupsForOwnerlessScan(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/findings/ingest":
			if r.Method != http.MethodPost {
				t.Fatalf("ingest method = %s, want POST", r.Method)
			}
			if got := r.Header.Get(processingConsoleUserHeader); got != "" {
				t.Fatalf("%s = %q, want empty", processingConsoleUserHeader, got)
			}
			writeProcessingResponse(t, w)
		case "/api/v1/groups":
			t.Fatal("ownerless scan should not fetch global groups")
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	o := New(upstream.URL, "", "", "", "", "", "", nil, nil)
	resp, err := o.passportAfterFindings(
		context.Background(),
		models.ScanRequest{ScannerName: "semgrep", TargetPath: "/repo"},
		[]models.ProcessingFindingItem{{AssetID: "app.go", Identifier: "rule", Severity: "high"}},
		0,
	)
	if err != nil {
		t.Fatalf("passportAfterFindings: %v", err)
	}
	if len(resp.Groups) != 0 {
		t.Fatalf("groups len = %d, want 0", len(resp.Groups))
	}
}

func TestPassportAfterFindingsFetchesGroupsForConsoleUser(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/findings/ingest":
			if got := r.Header.Get(processingConsoleUserHeader); got != "42" {
				t.Fatalf("ingest %s = %q, want 42", processingConsoleUserHeader, got)
			}
			writeProcessingResponse(t, w)
		case "/api/v1/groups":
			if got := r.Header.Get(processingConsoleUserHeader); got != "42" {
				t.Fatalf("groups %s = %q, want 42", processingConsoleUserHeader, got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]models.GroupResponse{{
				ID:           7,
				GroupKey:     "u:42:cve:CVE-2024-0001",
				GroupingRule: "cve",
				SeverityMax:  "high",
				AssetsCount:  1,
				Status:       "open",
			}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	o := New(upstream.URL, "", "", "", "", "", "", nil, nil)
	resp, err := o.passportAfterFindings(
		context.Background(),
		models.ScanRequest{ScannerName: "semgrep", TargetPath: "/repo"},
		[]models.ProcessingFindingItem{{AssetID: "app.go", Identifier: "rule", Severity: "high"}},
		42,
	)
	if err != nil {
		t.Fatalf("passportAfterFindings: %v", err)
	}
	if len(resp.Groups) != 1 || resp.Groups[0].ID != 7 {
		t.Fatalf("groups = %#v, want one group with id 7", resp.Groups)
	}
}

func writeProcessingResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models.ProcessingResponse{
		RunID:                  1,
		FindingsReceived:       1,
		FindingsProcessed:      1,
		VulnerabilitiesCreated: 1,
		GroupsUpdated:          1,
	}); err != nil {
		t.Fatalf("write processing response: %v", err)
	}
}
