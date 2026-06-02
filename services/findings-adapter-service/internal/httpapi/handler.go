package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"mephi_vkr_asoc/services/findings-adapter-service/internal/adapt"
	"mephi_vkr_asoc/services/findings-adapter-service/internal/models"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/adapt/semgrep", h.handleSemgrep)
	mux.HandleFunc("/api/v1/adapt/gitleaks", h.handleGitleaks)
	mux.HandleFunc("/api/v1/adapt/trivy", h.handleTrivy)
	mux.HandleFunc("/api/v1/adapt/zap", h.handleZAP)
	mux.HandleFunc("/api/v1/adapt/auto", h.handleAuto)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSemgrep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
		return
	}
	findings, err := adapt.Semgrep(raw)
	h.writeAdapt(w, findings, err)
}

func (h *Handler) handleGitleaks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
		return
	}
	findings, err := adapt.Gitleaks(raw)
	h.writeAdapt(w, findings, err)
}

func (h *Handler) handleTrivy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
		return
	}
	findings, err := adapt.Trivy(raw)
	h.writeAdapt(w, findings, err)
}

func (h *Handler) handleZAP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	targetURL := strings.TrimSpace(r.URL.Query().Get("target_url"))
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
		return
	}
	findings, err := adapt.ZAP(raw, targetURL)
	h.writeAdapt(w, findings, err)
}

func (h *Handler) handleAuto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	targetURL := strings.TrimSpace(r.URL.Query().Get("target_url"))
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
		return
	}
	findings, err := adapt.Flexible(raw, targetURL)
	h.writeAdapt(w, findings, err)
}

func (h *Handler) writeAdapt(w http.ResponseWriter, findings []models.FindingItem, err error) {
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if findings == nil {
		findings = []models.FindingItem{}
	}
	writeJSON(w, http.StatusOK, models.AdaptResponse{Findings: findings})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
