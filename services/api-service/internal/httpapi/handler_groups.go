package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"mephi_vkr_asoc/services/api-service/internal/models"
)

func (h *Handler) handleGroupsRoute(w http.ResponseWriter, r *http.Request) {
	if !h.ensureConsoleProductReportAccess(w, r) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/groups")
	path = strings.Trim(path, "/")
	if path == "" {
		h.proxyGETToProcessing(w, r, "/api/v1/groups")
		return
	}
	segments := strings.Split(path, "/")
	groupIDRaw := segments[0]
	if len(segments) == 2 && segments[1] == "jira-ticket" {
		h.handleGroupJiraTicket(w, r, groupIDRaw)
		return
	}
	if len(segments) == 1 && r.Method == http.MethodPatch {
		h.proxyPATCHGroupStatus(w, r, groupIDRaw)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (h *Handler) proxyPATCHGroupStatus(w http.ResponseWriter, r *http.Request, groupIDRaw string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	target := strings.TrimRight(h.processingURL, "/") + "/api/v1/groups/" + groupIDRaw
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPatch, target, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if uid, ok := ConsoleUserFromRequest(r); ok {
		req.Header.Set(HeaderConsoleUserID, strconv.FormatInt(uid, 10))
	}
	if cp := strings.TrimSpace(r.URL.Query().Get("console_product_id")); cp != "" {
		req.URL.RawQuery = "console_product_id=" + cp
	}
	resp, err := h.httpUpstream.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	copyUpstreamResponse(w, resp)
}

func (h *Handler) handleGroupJiraTicket(w http.ResponseWriter, r *http.Request, groupIDRaw string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	groupID, err := strconv.ParseInt(groupIDRaw, 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid group id"})
		return
	}
	group, err := h.fetchGroupForUser(r, groupID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if group.Status == "false_positive" || group.Status == "risk_accepted" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "group is closed; reopen before creating a Jira task"})
		return
	}
	jiraCtx, err := h.fetchGroupJiraContext(r, groupID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	ticket, err := h.orchestrator.CreateGroupTicket(r.Context(), buildTicketRequest(group, jiraCtx))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (h *Handler) fetchGroupForUser(r *http.Request, groupID int64) (models.GroupResponse, error) {
	limit := 500
	target := strings.TrimRight(h.processingURL, "/") + "/api/v1/groups?limit=" + strconv.Itoa(limit) + "&status=all"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		return models.GroupResponse{}, err
	}
	if uid, ok := ConsoleUserFromRequest(r); ok {
		req.Header.Set(HeaderConsoleUserID, strconv.FormatInt(uid, 10))
	}
	if qs := r.URL.Query().Get("console_product_id"); strings.TrimSpace(qs) != "" {
		req.URL.RawQuery = "limit=" + strconv.Itoa(limit) + "&console_product_id=" + qs
	}
	resp, err := h.httpUpstream.Do(req)
	if err != nil {
		return models.GroupResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return models.GroupResponse{}, fmt.Errorf("processing-service groups returned status %d", resp.StatusCode)
	}
	var groups []models.GroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return models.GroupResponse{}, err
	}
	for _, g := range groups {
		if g.ID == groupID {
			return g, nil
		}
	}
	return models.GroupResponse{}, errGroupNotFound
}

func buildTicketRequest(group models.GroupResponse, ctx models.GroupJiraContext) models.TicketRequest {
	req := models.TicketRequest{
		GroupID:        group.ID,
		GroupKey:       group.GroupKey,
		Severity:       group.SeverityMax,
		AssetsCount:    group.AssetsCount,
		CorrelationRef: group.GroupKey,
	}
	if len(ctx.Vulnerabilities) > 0 {
		req.Vulnerabilities = ctx.Vulnerabilities
	}
	return req
}

func (h *Handler) fetchGroupJiraContext(r *http.Request, groupID int64) (models.GroupJiraContext, error) {
	target := strings.TrimRight(h.processingURL, "/") + "/api/v1/groups/" + strconv.FormatInt(groupID, 10) + "/jira-context"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		return models.GroupJiraContext{}, err
	}
	if uid, ok := ConsoleUserFromRequest(r); ok {
		req.Header.Set(HeaderConsoleUserID, strconv.FormatInt(uid, 10))
	}
	q := r.URL.Query()
	if cp := strings.TrimSpace(q.Get("console_product_id")); cp != "" {
		req.URL.RawQuery = "console_product_id=" + cp
	}
	resp, err := h.httpUpstream.Do(req)
	if err != nil {
		return models.GroupJiraContext{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return models.GroupJiraContext{}, errGroupNotFound
	}
	if resp.StatusCode >= 300 {
		return models.GroupJiraContext{}, fmt.Errorf("processing-service jira-context returned status %d", resp.StatusCode)
	}
	var ctx models.GroupJiraContext
	if err := json.NewDecoder(resp.Body).Decode(&ctx); err != nil {
		return models.GroupJiraContext{}, err
	}
	return ctx, nil
}

var errGroupNotFound = errNotFound("group not found or forbidden")

type errNotFound string

func (e errNotFound) Error() string { return string(e) }

func copyUpstreamResponse(w http.ResponseWriter, resp *http.Response) {
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
