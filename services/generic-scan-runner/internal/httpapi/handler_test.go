package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandleRunRejectsTargetOutsideAllowedRoots(t *testing.T) {
	root := t.TempDir()
	h := &Handler{
		ExecTimeout:      time.Second,
		AllowedScanRoots: []string{root},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(map[string]string{
		"runner_command": `printf '%s' '[]'`,
		"target_path":    "/var/run/secrets/kubernetes.io/serviceaccount",
		"scanner_id":     "custom",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/run", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "outside allowed scan roots") {
		t.Fatalf("expected confinement error, got %s", rr.Body.String())
	}
}

func TestHandleRunAllowsTargetUnderAllowedRoot(t *testing.T) {
	root := t.TempDir()
	scanDir := filepath.Join(root, "proj")
	if err := os.MkdirAll(scanDir, 0o750); err != nil {
		t.Fatal(err)
	}

	h := &Handler{
		ExecTimeout:      time.Second,
		AllowedScanRoots: []string{root},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(map[string]string{
		"runner_command": `printf '%s' '[{"ok":true}]'`,
		"target_path":    scanDir,
		"scanner_id":     "custom",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/run", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !json.Valid(rr.Body.Bytes()) {
		t.Fatalf("expected JSON stdout, got %s", rr.Body.String())
	}
}

func TestHandleRunDefaultsEmptyTargetToTmpWhenAllowed(t *testing.T) {
	h := &Handler{
		ExecTimeout:      time.Second,
		AllowedScanRoots: []string{"/tmp"},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(map[string]string{
		"runner_command": `printf '%s' '{"findings":[]}'`,
		"scanner_id":     "custom",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/run", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
