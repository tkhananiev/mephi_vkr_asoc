package httpapi

import (
	"net/http"
	"strings"

	"mephi_vkr_asoc/services/api-service/internal/scm"
)

func (h *Handler) handleConsoleRepositoryBranches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if _, ok := ConsoleUserFromRequest(r); !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console user JWT required"})
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("repository_url"))
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repository_url required"})
		return
	}
	branches, err := scm.ListRemoteBranches(r.Context(), raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"branches": branches})
}
