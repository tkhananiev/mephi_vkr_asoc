package integrationstore

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateScannerInvokeURLRejectsMetadataAndLoopback(t *testing.T) {
	t.Parallel()
	cases := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/",
		"http://127.0.0.1:8087/api/v1/run",
		"http://localhost:8087/api/v1/run",
		"http://user:pass@example.com/run",
		"ftp://example.com/run",
		"not-a-url",
	}
	for _, raw := range cases {
		if err := ValidateScannerInvokeURL(raw); err == nil {
			t.Fatalf("expected reject for %q", raw)
		}
	}
}

func TestValidateScannerInvokeURLAllowsClusterAndPublic(t *testing.T) {
	t.Parallel()
	if err := ValidateScannerInvokeURL("http://10.43.0.12:8087/api/v1/run"); err != nil {
		t.Fatalf("cluster IP should be allowed: %v", err)
	}
	if err := ValidateScannerInvokeURL("https://example.com/api/v1/run"); err != nil {
		t.Fatalf("public host should be allowed: %v", err)
	}
}

func TestValidateRejectsMetadataInvokeURL(t *testing.T) {
	t.Parallel()
	err := Validate(Item{
		ID:               "meta",
		Kind:             "SAST",
		Title:            "Meta",
		Phase:            "ready",
		InputKind:        "filesystem",
		ScannerName:      "meta",
		ScannerInvokeURL: "http://169.254.169.254/",
	}, map[string]struct{}{})
	if err == nil {
		t.Fatal("expected Validate to reject metadata invoke URL")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvokeHTTPClientRejectsRedirects(t *testing.T) {
	t.Parallel()
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stolen":true}`))
	}))
	defer final.Close()

	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redir.Close()

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("scanner_invoke_url redirects are not allowed")
		},
	}
	resp, err := client.Post(redir.URL, "application/json", strings.NewReader(`{}`))
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected redirect to fail")
	}
	if !strings.Contains(err.Error(), "redirects are not allowed") {
		t.Fatalf("expected redirect error, got %v", err)
	}
}
