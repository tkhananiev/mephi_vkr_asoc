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
)

type Handler struct {
	ExecTimeout time.Duration
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
	ctx, cancel := context.WithTimeout(r.Context(), h.ExecTimeout)
	defer cancel()

	expanded := expandCommand(cmdStr, body)
	workDir := strings.TrimSpace(body.TargetPath)
	if workDir == "" {
		workDir = "/tmp"
	}

	log.Printf("generic-scan-runner: scanner_id=%q dir=%q cmd=%q", strings.TrimSpace(body.ScannerID), workDir, expanded)

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", expanded)
	cmd.Dir = workDir
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
		"{target_path}", shellQuote(strings.TrimSpace(body.TargetPath)),
		"{TARGET_PATH}", shellQuote(strings.TrimSpace(body.TargetPath)),
		"{scanner_name}", shellQuote(strings.TrimSpace(body.ScannerName)),
		"{SCANNER_NAME}", shellQuote(strings.TrimSpace(body.ScannerName)),
		"{git_repository_url}", shellQuote(strings.TrimSpace(body.GitRepositoryURL)),
		"{GIT_REPOSITORY_URL}", shellQuote(strings.TrimSpace(body.GitRepositoryURL)),
		"{git_repository_ref}", shellQuote(strings.TrimSpace(body.GitRepositoryRef)),
		"{GIT_REPOSITORY_REF}", shellQuote(strings.TrimSpace(body.GitRepositoryRef)),
		"{semgrep_config}", shellQuote(strings.TrimSpace(body.SemgrepConfig)),
		"{SEMGREP_CONFIG}", shellQuote(strings.TrimSpace(body.SemgrepConfig)),
		"{scanner_id}", shellQuote(strings.TrimSpace(body.ScannerID)),
		"{SCANNER_ID}", shellQuote(strings.TrimSpace(body.ScannerID)),
	}
	r := strings.NewReplacer(repl...)
	return r.Replace(tpl)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
