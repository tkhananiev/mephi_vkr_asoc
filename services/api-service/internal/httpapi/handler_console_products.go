package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"mephi_vkr_asoc/services/api-service/internal/products"
)

func (h *Handler) handleConsoleProductsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.productStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "products store unavailable (set APP_POSTGRES_DSN)"})
		return
	}
	uid, ok := ConsoleUserFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console user JWT required"})
		return
	}
	rows, err := h.productStore.ListByOwner(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

type createProductBody struct {
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	RepositoryURL          string   `json:"repository_url"`
	RepositoryRef          string   `json:"repository_ref"`
	RepositoryBranchRefs   []string `json:"repository_branch_refs"`
	RepositorySubdirectory string   `json:"repository_subdirectory"`
	ScanTargetPath         string   `json:"scan_target_path"`
}

func (h *Handler) handleConsoleProductsCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.productStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "products store unavailable (set APP_POSTGRES_DSN)"})
		return
	}
	uid, ok := ConsoleUserFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console user JWT required"})
		return
	}
	var body createProductBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	row, err := h.productStore.Create(r.Context(), uid, products.CreateInput{
		Name:                   name,
		Description:            body.Description,
		RepositoryURL:          strings.TrimSpace(body.RepositoryURL),
		RepositoryRef:          body.RepositoryRef,
		RepositoryBranchRefs:   body.RepositoryBranchRefs,
		RepositorySubdirectory: body.RepositorySubdirectory,
		ScanTargetPath:         body.ScanTargetPath,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func parseConsoleProductPathID(path string) (int64, bool) {
	p := strings.TrimPrefix(path, "/api/v1/console/products/")
	p = strings.Trim(p, "/")
	if p == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(p, 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func (h *Handler) handleConsoleProductByPath(w http.ResponseWriter, r *http.Request) {
	id, ok := parseConsoleProductPathID(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	switch r.Method {
	case http.MethodPatch:
		h.handleConsoleProductPatch(w, r, id)
	case http.MethodDelete:
		h.handleConsoleProductDelete(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *Handler) handleConsoleProductPatch(w http.ResponseWriter, r *http.Request, id int64) {
	if h.productStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "products store unavailable (set APP_POSTGRES_DSN)"})
		return
	}
	uid, ok := ConsoleUserFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console user JWT required"})
		return
	}
	var body createProductBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	row, err := h.productStore.Update(r.Context(), uid, id, products.CreateInput{
		Name:                   name,
		Description:            body.Description,
		RepositoryURL:          strings.TrimSpace(body.RepositoryURL),
		RepositoryRef:          body.RepositoryRef,
		RepositoryBranchRefs:   body.RepositoryBranchRefs,
		RepositorySubdirectory: body.RepositorySubdirectory,
		ScanTargetPath:         body.ScanTargetPath,
	})
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) handleConsoleProductDelete(w http.ResponseWriter, r *http.Request, id int64) {
	if h.productStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "products store unavailable (set APP_POSTGRES_DSN)"})
		return
	}
	uid, ok := ConsoleUserFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console user JWT required"})
		return
	}
	err := h.productStore.Delete(r.Context(), uid, id)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
