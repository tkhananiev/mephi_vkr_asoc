package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mephi_vkr_asoc/services/api-service/internal/auth"
)

func TestGroupsRejectAPIKeyWithoutUserScope(t *testing.T) {
	var upstreamHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	handler := New(nil, "", "", testJWTSecret(), nil, nil, server.URL, nil)
	mux := http.NewServeMux()
	handler.Register(mux)
	wrapped := WithAPIKeyOrUserJWT("shared-key", testJWTSecret(), mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	req.Header.Set("X-API-Key", "shared-key")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d with body %s", rec.Code, rec.Body.String())
	}
	if upstreamHit {
		t.Fatal("API-key request was proxied without a user/admin scope")
	}
}

func TestGroupsForwardConsoleUserScope(t *testing.T) {
	var gotOwnerHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOwnerHeader = r.Header.Get(HeaderConsoleUserID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	secret := testJWTSecret()
	token, err := auth.Issue(secret, 42, "user@example.test", "User", "user", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(nil, "", "", secret, nil, nil, server.URL, nil)
	mux := http.NewServeMux()
	handler.Register(mux)
	wrapped := WithAPIKeyOrUserJWT("shared-key", secret, mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", rec.Code, rec.Body.String())
	}
	if gotOwnerHeader != "42" {
		t.Fatalf("expected owner header 42, got %q", gotOwnerHeader)
	}
}

func testJWTSecret() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}
