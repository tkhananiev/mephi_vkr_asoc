package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"mephi_vkr_asoc/services/trivy-sca-service/internal/runner"
	"mephi_vkr_asoc/services/trivy-sca-service/internal/workspace"
)

type Handler struct {
	runner                *runner.Runner
	defaultScanTargetPath string
	gitWorkspaceRoot      string
}

func New(r *runner.Runner, defaultScanTargetPath, gitWorkspaceRoot string) *Handler {
	return &Handler{
		runner:                r,
		defaultScanTargetPath: strings.TrimSpace(defaultScanTargetPath),
		gitWorkspaceRoot:      strings.TrimSpace(gitWorkspaceRoot),
	}
}

type scanRequest struct {
	TargetPath       string `json:"target_path"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`
	GitRepositoryRef string `json:"git_repository_ref,omitempty"`
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/scan", h.handleScan)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	gitURL := strings.TrimSpace(req.GitRepositoryURL)
	var cleanup func()

	targetPath := strings.TrimSpace(req.TargetPath)

	if gitURL != "" {
		workRoot := h.gitWorkspaceRoot
		subdir := strings.TrimSpace(targetPath)
		sd, clr, err := workspace.PrepareGitWorkspace(r.Context(), workRoot, gitURL, req.GitRepositoryRef, subdir)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		cleanup = clr
		defer cleanup()
		targetPath = sd
		log.Printf("trivy-sca-service: cloned %s → scan dir %s", gitURL, sd)
	} else {
		if targetPath == "" {
			targetPath = h.defaultScanTargetPath
		}
	}
	if targetPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target_path required (or set APP_DEFAULT_SCAN_TARGET_PATH) unless git_repository_url is set"})
		return
	}

	payload, err := h.runner.Run(r.Context(), targetPath)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
