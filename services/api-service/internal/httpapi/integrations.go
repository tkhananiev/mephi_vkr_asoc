package httpapi

import (
	"net/http"

	"mephi_vkr_asoc/services/api-service/internal/integrationstore"
)

type integrationsResponse struct {
	Integrations []integrationstore.Item `json:"integrations"`
}

func (h *Handler) handleIntegrationsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, integrationsResponse{Integrations: h.integrationStore.ListMerged()})
}
