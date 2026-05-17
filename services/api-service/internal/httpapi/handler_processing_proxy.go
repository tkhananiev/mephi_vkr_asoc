package httpapi

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) handleGroupsProxy(w http.ResponseWriter, r *http.Request) {
	h.proxyGETToProcessing(w, r, "/api/v1/groups")
}

func (h *Handler) handleReportVulnerabilitiesProxy(w http.ResponseWriter, r *http.Request) {
	h.proxyGETToProcessing(w, r, "/api/v1/report/vulnerabilities")
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
