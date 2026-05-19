package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) handleGroupsProxy(w http.ResponseWriter, r *http.Request) {
	if !h.ensureConsoleProductReportAccess(w, r) {
		return
	}
	h.proxyGETToProcessing(w, r, "/api/v1/groups")
}

func (h *Handler) handleReportVulnerabilitiesProxy(w http.ResponseWriter, r *http.Request) {
	if !h.ensureConsoleProductReportAccess(w, r) {
		return
	}
	h.proxyGETToProcessing(w, r, "/api/v1/report/vulnerabilities")
}

func parseConsoleProductQueryParam(r *http.Request) (*int64, error) {
	q := strings.TrimSpace(r.URL.Query().Get("console_product_id"))
	if q == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(q, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid console_product_id")
	}
	return &id, nil
}

// ensureConsoleProductReportAccess — при ?console_product_id= нужен JWT пользователя и владение продуктом.
func (h *Handler) ensureConsoleProductReportAccess(w http.ResponseWriter, r *http.Request) bool {
	cp, err := parseConsoleProductQueryParam(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return false
	}
	if cp == nil {
		return true
	}
	uid, ok := ConsoleUserFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id requires console user JWT"})
		return false
	}
	if h.productStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "products store unavailable (set APP_POSTGRES_DSN)"})
		return false
	}
	owned, err := h.productStore.ProductOwnedBy(r.Context(), *cp, uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return false
	}
	if !owned {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id not found or forbidden"})
		return false
	}
	return true
}

func (h *Handler) proxyGETToProcessing(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	target := strings.TrimRight(h.processingURL, "/") + path
	if qs := r.URL.RawQuery; qs != "" {
		target += "?" + qs
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if uid, ok := ConsoleUserFromRequest(r); ok {
		req.Header.Set(HeaderConsoleUserID, strconv.FormatInt(uid, 10))
	}
	resp, err := h.httpUpstream.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func defaultHTTPUpstream() *http.Client {
	return &http.Client{Timeout: 2 * time.Minute}
}
