package httpapi

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type mockIssue struct {
	ID        int64
	Key       string
	Summary   string
	CreatedAt time.Time
}

type Handler struct {
	counter         atomic.Int64
	publicIssueBase string
	mu              sync.RWMutex
	issues          []mockIssue
}

func New(publicIssueBase string) *Handler {
	b := strings.TrimRight(strings.TrimSpace(publicIssueBase), "/")
	if b == "" {
		b = "http://localhost:8090"
	}
	return &Handler{publicIssueBase: b}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/console", h.handleConsole)
	mux.HandleFunc("/browse/", h.handleBrowse)
	mux.HandleFunc("/rest/api/2/issue", h.handleCreateIssue)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleConsole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.mu.RLock()
	copyIssues := append([]mockIssue(nil), h.issues...)
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><meta charset="utf-8"><title>Jira mock — тикеты</title>`)
	b.WriteString(`<style>body{font-family:sans-serif;margin:1.5rem;} table{border-collapse:collapse;} th,td{border:1px solid #ccc;padding:.4rem .6rem;text-align:left;} th{background:#f0f0f0;} tr:nth-child(even){background:#fafafa;} .hint{color:#666;font-size:.9rem;}</style>`)
	b.WriteString(`<body><h1>Заглушка Jira (mock)</h1><p class="hint">Список задач, созданных через REST (<code>POST /rest/api/2/issue</code>). Данные только в памяти процесса.</p>`)
	if len(copyIssues) == 0 {
		b.WriteString(`<p>Пока нет тикетов — запусти сценарий сканирования с созданием задач.</p></body>`)
		_, _ = w.Write([]byte(b.String()))
		return
	}
	b.WriteString(`<table><thead><tr><th>Ключ</th><th>Кратко</th><th>Создан</th></tr></thead><tbody>`)
	for i := len(copyIssues) - 1; i >= 0; i-- {
		iss := copyIssues[i]
		b.WriteString(`<tr><td><a href="/browse/`)
		b.WriteString(html.EscapeString(iss.Key))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(iss.Key))
		b.WriteString(`</a></td><td>`)
		b.WriteString(html.EscapeString(iss.Summary))
		b.WriteString(`</td><td>`)
		b.WriteString(html.EscapeString(iss.CreatedAt.Format(time.RFC3339)))
		b.WriteString(`</td></tr>`)
	}
	b.WriteString(`</tbody></table></body>`)
	_, _ = w.Write([]byte(b.String()))
}

func (h *Handler) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/browse/")
	key = strings.TrimSpace(key)
	if key == "" {
		http.NotFound(w, r)
		return
	}
	summary := ""
	h.mu.RLock()
	for i := len(h.issues) - 1; i >= 0; i-- {
		if h.issues[i].Key == key {
			summary = h.issues[i].Summary
			break
		}
	}
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if summary != "" {
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html><meta charset="utf-8"><title>%s</title><body><p><a href="/console">← Все тикеты</a></p><h1>%s</h1><p>%s</p></body>`,
			html.EscapeString(key), html.EscapeString(key), html.EscapeString(summary))
		return
	}
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><meta charset="utf-8"><title>%s</title><body><p><a href="/console">← Все тикеты</a></p><h1>%s</h1><p>Задача не найдена в памяти mock (возможно, перезапуск контейнера).</p></body>`,
		html.EscapeString(key), html.EscapeString(key))
}

func (h *Handler) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
		return
	}
	summary := extractSummaryFromJiraBody(bodyBytes)
	if summary == "" {
		summary = "(без summary в теле запроса)"
	}

	id := h.counter.Add(1)
	key := "ASOC-" + itoa(id)

	h.mu.Lock()
	h.issues = append(h.issues, mockIssue{
		ID:        id,
		Key:       key,
		Summary:   summary,
		CreatedAt: time.Now().UTC(),
	})
	h.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":   itoa(id),
		"key":  key,
		"self": h.publicIssueBase + "/browse/" + key,
	})
}

func extractSummaryFromJiraBody(raw []byte) string {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return ""
	}
	fieldsRaw, ok := root["fields"]
	if !ok {
		return ""
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(fieldsRaw, &fields); err != nil {
		return ""
	}
	sumRaw, ok := fields["summary"]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(sumRaw, &s); err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func itoa(value int64) string {
	return fmt.Sprintf("%d", value)
}
