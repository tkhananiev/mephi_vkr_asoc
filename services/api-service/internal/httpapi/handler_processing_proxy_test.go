package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mephi_vkr_asoc/services/api-service/internal/auth"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func testAPIHandler(t *testing.T, processingURL string) http.Handler {
	t.Helper()
	h := New(nil, "", "", []byte(testJWTSecret), nil, nil, processingURL, nil)
	mux := http.NewServeMux()
	h.Register(mux)
	return WithAPIKeyOrUserJWT("test-api-key", []byte(testJWTSecret), mux)
}

func testUserJWT(t *testing.T, userID int64) string {
	t.Helper()
	token, err := auth.Issue([]byte(testJWTSecret), userID, "user@example.test", "Test User", "user", time.Hour)
	if err != nil {
		t.Fatalf("issue jwt: %v", err)
	}
	return token
}

func TestGroupsAndReportsRequireConsoleUserJWT(t *testing.T) {
	upstreamHit := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusTeapot)
	}))
	defer upstream.Close()

	handler := testAPIHandler(t, upstream.URL)
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "groups list", method: http.MethodGet, path: "/api/v1/groups"},
		{name: "report list", method: http.MethodGet, path: "/api/v1/report/vulnerabilities"},
		{name: "group status", method: http.MethodPatch, path: "/api/v1/groups/42", body: `{"status":"false_positive"}`},
		{name: "jira ticket", method: http.MethodPost, path: "/api/v1/groups/42/jira-ticket"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstreamHit = false
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("X-API-Key", "test-api-key")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if upstreamHit {
				t.Fatal("processing upstream was called for API-key-only request")
			}
		})
	}
}

func TestGroupsProxyForwardsConsoleUserHeader(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(HeaderConsoleUserID); got != "42" {
			t.Fatalf("%s = %q, want 42", HeaderConsoleUserID, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "group_key": "u:42:cve:CVE-2024-0001", "grouping_rule": "cve", "severity_max": "high", "assets_count": 1, "status": "open"},
		})
	}))
	defer upstream.Close()

	handler := testAPIHandler(t, upstream.URL)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	req.Header.Set("Authorization", "Bearer "+testUserJWT(t, 42))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
