package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReferenceSyncProxyRequiresAPIAuth(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusTeapot)
	}))
	defer upstream.Close()

	mux := http.NewServeMux()
	h := New(nil, "", "", nil, nil, nil, upstream.URL, "", nil)
	h.Register(mux)
	root := WithAPIKeyOrUserJWT("sync-secret", nil, mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/nvd", nil)
	rr := httptest.NewRecorder()
	root.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if upstreamCalled {
		t.Fatal("unauthorized sync request reached reference-data-service")
	}
}

func TestReferenceSyncProxyForwardsAuthorizedRequests(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.RequestURI(); got != "/api/v1/sync/nvd?full=1" {
			t.Errorf("request uri = %q, want /api/v1/sync/nvd?full=1", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "" {
			t.Errorf("X-API-Key was forwarded to reference-data-service: %q", got)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode forwarded body: %v", err)
		}
		if got := payload["mode"]; got != "full" {
			t.Errorf("forwarded body mode = %q, want full", got)
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
	}))
	defer upstream.Close()

	mux := http.NewServeMux()
	h := New(nil, "", "", nil, nil, nil, upstream.URL, "", nil)
	h.Register(mux)
	root := WithAPIKeyOrUserJWT("sync-secret", nil, mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/nvd?full=1", strings.NewReader(`{"mode":"full"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "sync-secret")
	rr := httptest.NewRecorder()
	root.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusAccepted, rr.Body.String())
	}
	if got := strings.TrimSpace(rr.Body.String()); got != `{"status":"accepted"}` {
		t.Fatalf("body = %q, want accepted status JSON", got)
	}
}
