package httpapi

import (
	"encoding/json"
	"net/http"

	"mephi_vkr_asoc/services/api-service/internal/integrationstore"
)

type adminIntegrationsResponse struct {
	Entries           []integrationstore.AdminRow `json:"entries"`
	OverlayPersistent bool                        `json:"overlay_persistent"`
}

type adminIntegrationsPutBody struct {
	Additional []integrationstore.Item `json:"additional"`
}

func (h *Handler) handleAdminIntegrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPut:
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if _, ok := h.requireAdminRole(r); !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin bearer token required"})
		return
	}

	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, adminIntegrationsResponse{
			Entries:           h.integrationStore.AdminList(),
			OverlayPersistent: h.integrationStore.HasPersistentOverlay(),
		})
		return
	}

	var body adminIntegrationsPutBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if body.Additional == nil {
		body.Additional = []integrationstore.Item{}
	}
	if err := h.integrationStore.ReplaceAdditional(body.Additional); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
