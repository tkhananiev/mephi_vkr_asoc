package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
)

type Handler struct {
	counter atomic.Int64
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/rest/api/2/issue", h.handleCreateIssue)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	id := h.counter.Add(1)
	key := "ASPM-" + itoa(id)
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":   itoa(id),
		"key":  key,
		"self": "http://jira-mock:8090/browse/" + key,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func itoa(value int64) string {
	return fmt.Sprintf("%d", value)
}
