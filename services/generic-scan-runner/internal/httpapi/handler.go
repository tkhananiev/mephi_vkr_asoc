package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"mephi_vkr_asoc/services/generic-scan-runner/internal/workspace"
)

type Handler struct {
	ExecTimeout      time.Duration
	AllowedScanRoots []string
}

type runRequest struct {
	TargetPath       string `json:"target_path"`
	ScannerName      string `json:"scanner_name"`
	SemgrepConfig    string `json:"semgrep_config,omitempty"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`
	GitRepositoryRef string `json:"git_repository_ref,omitempty"`
	ScannerID        string `json:"scanner_id"`
	RunnerCommand    string `json:"runner_command"`
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/run", h.handleRun)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body runRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	cmdStr := strings.TrimSpace(body.RunnerCommand)
	if cmdStr == "" {
		writeErr(w, http.StatusBadRequest, "runner_command is required")
		return
	}
	if h.ExecTimeout <= 0 {
		h.ExecTimeout = 15 * time.Minute
	}

	workDir := strings.TrimSpace(body.TargetPath)
	if workDir == "" {
		workDir = "/tmp"
	}
	confined, err := workspace.AssertPathUnderAnyRoot(workDir, h.AllowedScanRoots)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	body.TargetPath = confined

	ctx, cancel := context.WithTimeout(r.Context(), h.ExecTimeout)
	defer cancel()

	expanded := expandCommand(cmdStr, body)

	log.Printf("generic-scan-runner: scanner_id=%q dir=%q cmd=%q", strings.TrimSpace(body.ScannerID), confined, expanded)

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", expanded)
	cmd.Dir = confined
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("command failed: %v; output=%s", err, truncate(string(out), 8000))
		log.Printf("generic-scan-runner: %s", msg)
		writeErr(w, http.StatusBadGateway, msg)
		return
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		writeErr(w, http.StatusBadGateway, "scanner produced empty stdout")
		return
	}
	if !json.Valid([]byte(raw)) {
		writeErr(w, http.StatusBadGateway, "scanner stdout is not valid JSON (first 500 chars): "+truncate(raw, 500))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(raw))
}

func expandCommand(tpl string, body runRequest) string {
	repl := []string{
		"{target_path}", strings.TrimSpace(body.TargetPath),
		"{TARGET_PATH}", strings.TrimSpace(body.TargetPath),
		"{scanner_name}", strings.TrimSpace(body.ScannerName),
		"{SCANNER_NAME}", strings.TrimSpace(body.ScannerName),
		"{git_repository_url}", strings.TrimSpace(body.GitRepositoryURL),
		"{GIT_REPOSITORY_URL}", strings.TrimSpace(body.GitRepositoryURL),
		"{git_repository_ref}", strings.TrimSpace(body.GitRepositoryRef),
		"{GIT_REPOSITORY_REF}", strings.TrimSpace(body.GitRepositoryRef),
		"{semgrep_config}", strings.TrimSpace(body.SemgrepConfig),
		"{SEMGREP_CONFIG}", strings.TrimSpace(body.SemgrepConfig),
		"{scanner_id}", strings.TrimSpace(body.ScannerID),
		"{SCANNER_ID}", strings.TrimSpace(body.ScannerID),
	}
	r := strings.NewReplacer(repl...)
	return r.Replace(tpl)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
