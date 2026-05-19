package integrationstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// Item — публичное описание интеграции (совпадает с JSON для GET /api/v1/integrations).
type Item struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Phase        string   `json:"phase"`
	Enabled      bool     `json:"enabled"`
	InputKind    string   `json:"input_kind"` // filesystem | lockfile | http
	ScannerName  string   `json:"scanner_name,omitempty"`
	APIScanPath  string   `json:"api_scan_path,omitempty"`
	ConsolePath  string   `json:"console_path,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Note         string   `json:"note,omitempty"`
	// NetworkHost — краткая сводка или устаревшее положение; может синхронизироваться с hostname/ip:port.
	NetworkHost string `json:"network_host,omitempty"`
	// NetworkHostname, NetworkIP, NetworkPort — сетевое расположение (справочно для оператора).
	NetworkHostname string `json:"network_hostname,omitempty"`
	NetworkIP       string `json:"network_ip,omitempty"`
	NetworkPort     string `json:"network_port,omitempty"`
	// ScannerInvokeURL — полный URL POST запуска скана (доп. сканеры из админ-каталога).
	ScannerInvokeURL string `json:"scanner_invoke_url,omitempty"`
	// InvokeHint — как запускать вручную: CLI, поля POST и т.д. (только документация для админов).
	InvokeHint string `json:"invoke_hint,omitempty"`
	// RunnerCommand — шаблон shell для generic-scan-runner (плейсхолдеры {target_path}, {git_repository_url}, …).
	RunnerCommand string `json:"runner_command,omitempty"`
	// InvokePayloadTemplate — пример/шаблон тела POST (документация и настройка для админов).
	InvokePayloadTemplate string `json:"invoke_payload_template,omitempty"`
}

// builtinCatalog жёстко зашит в api-service; доп. строки — только через overlay (PUT админки).
var builtinCatalog = []Item{
	{
		ID:           "semgrep",
		Kind:         "SAST",
		Title:        "Semgrep",
		Summary:      "",
		Phase:        "ready",
		Enabled:      true,
		InputKind:    "filesystem",
		ScannerName:  "semgrep",
		APIScanPath:  "/api/v1/scans",
		ConsolePath:  "/app/scan/semgrep",
		Capabilities: []string{"sast", "filesystem_target"},
	},
	{
		ID:           "gitleaks",
		Kind:         "SAST",
		Title:        "Gitleaks",
		Summary:      "Поиск секретов и чувствительной информации в исходном коде.",
		Phase:        "ready",
		Enabled:      true,
		InputKind:    "filesystem",
		ScannerName:  "gitleaks",
		APIScanPath:  "/api/v1/scans/gitleaks",
		ConsolePath:  "/app/scan/gitleaks",
		Capabilities: []string{"secrets", "filesystem_target"},
	},
}

func builtinIDs() map[string]struct{} {
	m := make(map[string]struct{}, len(builtinCatalog))
	for _, b := range builtinCatalog {
		m[b.ID] = struct{}{}
	}
	return m
}

type overlayFile struct {
	Version    int    `json:"version"`
	Additional []Item `json:"additional"`
}

type Store struct {
	mu         sync.RWMutex
	path       string // пустой — только память между рестартами
	additional []Item
}

// New загружает «дополнительные» сканеры с диска, если указан overlayPath и файл существует.
func New(overlayPath string) (*Store, error) {
	path := strings.TrimSpace(overlayPath)
	s := &Store{path: path}
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	var of overlayFile
	if err := json.Unmarshal(data, &of); err != nil {
		return nil, fmt.Errorf("integration overlay invalid json: %w", err)
	}
	reserved := builtinIDs()
	norm := normalizeItems(of.Additional)
	if err := validateAdditional(norm, reserved); err != nil {
		return nil, err
	}
	s.additional = norm
	return s, nil
}

func normalizeItems(xs []Item) []Item {
	out := make([]Item, len(xs))
	for i := range xs {
		out[i] = xs[i]
		out[i].ID = strings.TrimSpace(out[i].ID)
		out[i].Kind = strings.TrimSpace(out[i].Kind)
		out[i].Title = strings.TrimSpace(out[i].Title)
		out[i].Summary = strings.TrimSpace(out[i].Summary)
		out[i].Phase = strings.TrimSpace(strings.ToLower(out[i].Phase))
		out[i].InputKind = strings.TrimSpace(strings.ToLower(out[i].InputKind))
		out[i].ScannerName = strings.TrimSpace(out[i].ScannerName)
		out[i].APIScanPath = strings.TrimSpace(out[i].APIScanPath)
		out[i].ConsolePath = strings.TrimSpace(out[i].ConsolePath)
		out[i].Note = strings.TrimSpace(out[i].Note)
		out[i].NetworkHost = strings.TrimSpace(out[i].NetworkHost)
		out[i].NetworkHostname = strings.TrimSpace(out[i].NetworkHostname)
		out[i].NetworkIP = strings.TrimSpace(out[i].NetworkIP)
		out[i].NetworkPort = strings.TrimSpace(out[i].NetworkPort)
		out[i].ScannerInvokeURL = strings.TrimSpace(out[i].ScannerInvokeURL)
		out[i].InvokeHint = strings.TrimSpace(out[i].InvokeHint)
		out[i].RunnerCommand = strings.TrimSpace(out[i].RunnerCommand)
		out[i].InvokePayloadTemplate = strings.TrimSpace(out[i].InvokePayloadTemplate)
		sum := strings.TrimSpace(out[i].Summary)
		if sum == "" {
			sum = strings.TrimSpace(out[i].Title)
		}
		if sum == "" {
			sum = "-"
		}
		out[i].Summary = sum
		if len(out[i].Capabilities) == 0 {
			out[i].Capabilities = nil
		}
		if out[i].NetworkHost == "" && (out[i].NetworkHostname != "" || out[i].NetworkIP != "" || out[i].NetworkPort != "") {
			var nb strings.Builder
			if out[i].NetworkHostname != "" {
				nb.WriteString(out[i].NetworkHostname)
			}
			if out[i].NetworkIP != "" || out[i].NetworkPort != "" {
				if nb.Len() > 0 {
					nb.WriteString(" · ")
				}
				if out[i].NetworkIP != "" && out[i].NetworkPort != "" {
					nb.WriteString(out[i].NetworkIP)
					nb.WriteByte(':')
					nb.WriteString(out[i].NetworkPort)
				} else if out[i].NetworkIP != "" {
					nb.WriteString(out[i].NetworkIP)
				} else {
					nb.WriteByte(':')
					nb.WriteString(out[i].NetworkPort)
				}
			}
			out[i].NetworkHost = nb.String()
		}
	}
	return out
}

func validateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("id required")
	}
	for _, r := range id {
		switch {
		case unicode.IsLetter(r) && unicode.IsLower(r):
		case unicode.IsDigit(r):
		case r == '-' || r == '_':
		default:
			return fmt.Errorf("id %q: only lowercase letters, digits, - and _", id)
		}
	}
	return nil
}

// Validate проверка одной записи перед сохранением.
func Validate(it Item, reserved map[string]struct{}) error {
	if err := validateID(it.ID); err != nil {
		return err
	}
	it.ID = strings.TrimSpace(it.ID)
	if _, bad := reserved[it.ID]; bad {
		return fmt.Errorf("id %q is reserved (builtin scanner)", it.ID)
	}

	k := strings.TrimSpace(it.Kind)
	allowedKinds := map[string]struct{}{
		"SAST":       {},
		"SCA":        {},
		"DAST":       {},
		"MAST":       {},
		"Image Scan": {},
	}
	if _, ok := allowedKinds[k]; !ok {
		return fmt.Errorf("kind must be one of: SAST, DAST, MAST, SCA, Image Scan")
	}
	ph := strings.TrimSpace(strings.ToLower(it.Phase))
	if ph != "ready" && ph != "planned" {
		return fmt.Errorf("phase must be ready or planned")
	}
	ik := strings.TrimSpace(strings.ToLower(it.InputKind))
	if ik != "filesystem" && ik != "lockfile" && ik != "http" {
		return fmt.Errorf("input_kind must be filesystem, lockfile or http")
	}
	if strings.TrimSpace(it.Title) == "" {
		return errors.New("title required")
	}
	rc := strings.TrimSpace(it.RunnerCommand)
	if rc != "" && strings.TrimSpace(it.ScannerInvokeURL) == "" {
		return errors.New("scanner_invoke_url required when runner_command is set")
	}
	if ph == "ready" {
		sn := strings.TrimSpace(it.ScannerName)
		if sn == "" {
			return errors.New("scanner_name required when phase is ready")
		}
	}
	if u := strings.TrimSpace(it.ScannerInvokeURL); u != "" {
		parsed, err := url.Parse(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("scanner_invoke_url must be absolute URL with http or https scheme")
		}
	}
	return nil
}

func validateAdditional(xs []Item, reserved map[string]struct{}) error {
	seen := make(map[string]struct{}, len(xs))
	for _, it := range xs {
		if err := Validate(it, reserved); err != nil {
			return err
		}
		id := strings.TrimSpace(it.ID)
		if _, dup := seen[id]; dup {
			return fmt.Errorf("duplicate id %q in additional integrations", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (s *Store) persistLocked() error {
	if s.path == "" {
		return nil
	}
	of := overlayFile{Version: 1, Additional: s.additional}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(of); err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if dir != "." && dir != "/" {
		_ = os.MkdirAll(dir, 0750)
	}
	tmp, err := os.CreateTemp(dir, "integration-overlay-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.Write(buf.Bytes())
	closeErr := tmp.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// ListMerged — встроенные в фиксированном порядке, затем «дополнительные», отсортированные по id.
func (s *Store) ListMerged() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	add := append([]Item(nil), s.additional...)
	sort.SliceStable(add, func(i, j int) bool {
		return add[i].ID < add[j].ID
	})
	out := append(append([]Item{}, builtinCatalog...), add...)
	return out
}

// AdminRow — элемент ответа GET /api/v1/admin/integrations.
type AdminRow struct {
	Source      string `json:"source"` // builtin | additional
	Integration Item   `json:"integration"`
}

// AdminList для Atomic-admin UI.
func (s *Store) AdminList() []AdminRow {
	s.mu.RLock()
	defer s.mu.RUnlock()

	add := append([]Item(nil), s.additional...)
	sort.SliceStable(add, func(i, j int) bool {
		return add[i].ID < add[j].ID
	})
	rows := make([]AdminRow, 0, len(builtinCatalog)+len(add))
	for _, it := range builtinCatalog {
		rows = append(rows, AdminRow{Source: "builtin", Integration: it})
	}
	for _, it := range add {
		rows = append(rows, AdminRow{Source: "additional", Integration: it})
	}
	return rows
}

// HasPersistentOverlay true, если при старте задан путь к JSON overlay (монтирование тома сохранит доп. сканеры).
func (s *Store) HasPersistentOverlay() bool {
	return strings.TrimSpace(s.path) != ""
}

// ReplaceAdditional полностью заменяет «дополнительные» сканеры (не трогая встроенные).
func (s *Store) ReplaceAdditional(xs []Item) error {
	norm := normalizeItems(xs)
	reserved := builtinIDs()
	if err := validateAdditional(norm, reserved); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.additional = norm
	return s.persistLocked()
}

// LookupAdditional возвращает запись из дополнительного каталога по scanner_id (регистронезависимо).
func (s *Store) LookupAdditional(scannerID string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(scannerID))
	for _, it := range s.additional {
		if strings.ToLower(strings.TrimSpace(it.ID)) == key {
			return it, true
		}
	}
	return Item{}, false
}

// LookupDynamicInvoke возвращает URL POST, имя сканера и опционально runner_command (generic-scan-runner).
func (s *Store) LookupDynamicInvoke(scannerID string) (invokeURL string, scannerName string, runnerCommand string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(scannerID))
	for _, it := range s.additional {
		if strings.ToLower(strings.TrimSpace(it.ID)) != key {
			continue
		}
		u := strings.TrimSpace(it.ScannerInvokeURL)
		if u == "" {
			return "", "", "", false
		}
		sn := strings.TrimSpace(it.ScannerName)
		if sn == "" {
			sn = strings.TrimSpace(it.ID)
		}
		return u, sn, strings.TrimSpace(it.RunnerCommand), true
	}
	return "", "", "", false
}
