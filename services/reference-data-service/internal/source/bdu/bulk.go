package bdu

import (
	"archive/zip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"mephi_vkr_asoc/services/reference-data-service/internal/models"
)

// Потоковый разбор XML не загружает весь файл в память; xlsx читается построчно через excelize.
type BulkImporter struct {
	HTTP          *http.Client
	ZipURL        string
	XLSXURL       string
	ZipLocalPath  string // путь к vulxml.zip или к vulxml.xml; или file://…; если задан, не качается по HTTP
	XLSXLocalPath string
	BatchSize     int
}

func NewBulkImporter(httpClient *http.Client, zipURL, xlsxURL string, batchSize int, zipLocalPath, xlsxLocalPath string) *BulkImporter {
	if batchSize <= 0 {
		batchSize = 500
	}
	return &BulkImporter{
		HTTP:          httpClient,
		ZipURL:        strings.TrimSpace(zipURL),
		XLSXURL:       strings.TrimSpace(xlsxURL),
		ZipLocalPath:  strings.TrimSpace(zipLocalPath),
		XLSXLocalPath: strings.TrimSpace(xlsxLocalPath),
		BatchSize:     batchSize,
	}
}

func normalizeBDULocalRef(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "file:") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("file URL: %w", err)
		}
		if !strings.EqualFold(u.Scheme, "file") {
			return "", fmt.Errorf("expected file:// URL, got scheme %q", u.Scheme)
		}
		p := u.Path
		if p == "" {
			p = filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(raw, "file:"), "//"))
		}
		if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		return filepath.Clean(p), nil
	}
	return filepath.Clean(raw), nil
}

func (b *BulkImporter) Import(ctx context.Context, onBatch func([]models.SourceRecord) error) error {
	vulPath, vulCleanup, plainXML, err := b.resolveVulXMLSource(ctx)
	if err != nil {
		return err
	}
	if vulCleanup != nil {
		defer vulCleanup()
	}

	if plainXML {
		log.Printf("[bdu-bulk] phase vulxml — streaming file %q (до первых батчей в БД может уйти много времени на разбор большого XML)", vulPath)
		if err := b.streamVulXMLFromPlainFile(ctx, vulPath, onBatch); err != nil {
			return fmt.Errorf("vulxml: %w", err)
		}
	} else {
		log.Printf("[bdu-bulk] phase vulxml — reading zip %q", vulPath)
		if err := b.streamVulXMLFromZip(ctx, vulPath, onBatch); err != nil {
			return fmt.Errorf("vulxml: %w", err)
		}
	}

	log.Printf("[bdu-bulk] phase vulxml — завершена, переходим к vullist xlsx")

	xlsxPath, xlsxRm, err := b.resolveXLSX(ctx)
	if err != nil {
		return err
	}
	if xlsxRm != nil {
		defer xlsxRm()
	}

	log.Printf("[bdu-bulk] phase vullist — чтение %q", xlsxPath)

	if err := b.importVullistXLSX(ctx, xlsxPath, onBatch); err != nil {
		return fmt.Errorf("vullist: %w", err)
	}

	log.Printf("[bdu-bulk] все фазы BulkImporter завершены без ошибки")
	return nil
}

func (b *BulkImporter) resolveVulXMLSource(ctx context.Context) (path string, cleanup func(), plainXML bool, err error) {
	local, err := normalizeBDULocalRef(b.ZipLocalPath)
	if err != nil {
		return "", nil, false, fmt.Errorf("bdu bulk vulxml path: %w", err)
	}
	if local != "" {
		st, statErr := os.Stat(local)
		if statErr == nil {
			if st.IsDir() {
				return "", nil, false, fmt.Errorf("bdu bulk vulxml local path is a directory: %s", local)
			}
			if isPlainVulXMLFile(local) {
				return local, nil, true, nil
			}
			return local, nil, false, nil
		}
		if os.IsNotExist(statErr) {
			if strings.TrimSpace(b.ZipURL) != "" && b.HTTP != nil {
				log.Printf("[bdu-bulk] локальный vulxml %q не найден (пустой том?) — качаем по APP_BDU_VULXML_ZIP_URL", local)
			} else {
				return "", nil, false, fmt.Errorf(
					"bdu bulk vulxml: локальный файл %q не найден; задайте файлы на томе или URL/HTTP-клиент для загрузки",
					local,
				)
			}
		} else {
			return "", nil, false, fmt.Errorf("bdu bulk vulxml local %q: %w", local, statErr)
		}
	}
	if strings.TrimSpace(b.ZipURL) == "" {
		return "", nil, false, fmt.Errorf("bdu bulk: vulxml source not set (APP_BDU_VULXML_ZIP_PATH for .zip or .xml, or APP_BDU_VULXML_ZIP_URL)")
	}
	if b.HTTP == nil {
		return "", nil, false, fmt.Errorf("bdu bulk: http client is nil (needed for vulxml ZIP URL)")
	}
	tmp, err := b.downloadToTemp(ctx, b.ZipURL, "bdu-vulxml-*.zip")
	if err != nil {
		return "", nil, false, err
	}
	return tmp, func() { _ = os.Remove(tmp) }, false, nil
}

func isPlainVulXMLFile(absPath string) bool {
	return strings.EqualFold(filepath.Ext(absPath), ".xml")
}

func (b *BulkImporter) streamVulXMLFromPlainFile(ctx context.Context, xmlPath string, onBatch func([]models.SourceRecord) error) error {
	f, err := os.Open(xmlPath)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := xml.NewDecoder(f)
	dec.Strict = false
	return b.streamVulDecodeLoop(ctx, dec, onBatch)
}

func (b *BulkImporter) streamVulXMLFromZip(ctx context.Context, zipPath string, onBatch func([]models.SourceRecord) error) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	var xmlFile *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), "vulxml.xml") {
			xmlFile = f
			break
		}
	}
	if xmlFile == nil {
		return fmt.Errorf("vulxml.xml not found in zip")
	}

	rc, err := xmlFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	dec.Strict = false
	return b.streamVulDecodeLoop(ctx, dec, onBatch)
}

func (b *BulkImporter) streamVulDecodeLoop(ctx context.Context, dec *xml.Decoder, onBatch func([]models.SourceRecord) error) error {
	batch := make([]models.SourceRecord, 0, b.BatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		cp := make([]models.SourceRecord, len(batch))
		copy(cp, batch)
		batch = batch[:0]
		return onBatch(cp)
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		tok, err := dec.Token()
		if err == io.EOF {
			return flush()
		}
		if err != nil {
			return err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "vul" {
		
			if se.Name.Local == "vulnerabilities" {
				continue
			}
			if err := dec.Skip(); err != nil {
				return err
			}
			continue
		}
		var v vulXMLRecord
		if err := dec.DecodeElement(&v, &se); err != nil {
			return fmt.Errorf("decode <vul>: %w", err)
		}
		batch = append(batch, v.toSourceRecord())
		if len(batch) >= b.BatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
}

func (b *BulkImporter) resolveXLSX(ctx context.Context) (path string, cleanup func(), err error) {
	local, err := normalizeBDULocalRef(b.XLSXLocalPath)
	if err != nil {
		return "", nil, fmt.Errorf("bdu bulk xlsx path: %w", err)
	}
	if local != "" {
		st, statErr := os.Stat(local)
		if statErr == nil {
			if st.IsDir() {
				return "", nil, fmt.Errorf("bdu bulk xlsx local path is a directory: %s", local)
			}
			return local, nil, nil
		}
		if os.IsNotExist(statErr) {
			if strings.TrimSpace(b.XLSXURL) != "" && b.HTTP != nil {
				log.Printf("[bdu-bulk] локальный vullist %q не найден — качаем по APP_BDU_VULLIST_XLSX_URL", local)
			} else {
				return "", nil, fmt.Errorf(
					"bdu bulk xlsx: локальный файл %q не найден; задайте файлы на томе или URL/HTTP-клиент для загрузки",
					local,
				)
			}
		} else {
			return "", nil, fmt.Errorf("bdu bulk xlsx local %q: %w", local, statErr)
		}
	}
	if strings.TrimSpace(b.XLSXURL) == "" {
		return "", nil, fmt.Errorf("bdu bulk: xlsx source not set (APP_BDU_VULLIST_XLSX_PATH or APP_BDU_VULLIST_XLSX_URL)")
	}
	if b.HTTP == nil {
		return "", nil, fmt.Errorf("bdu bulk: http client is nil (needed for XLSX URL)")
	}
	tmp, err := b.downloadToTemp(ctx, b.XLSXURL, "bdu-vullist-*.xlsx")
	if err != nil {
		return "", nil, err
	}
	return tmp, func() { _ = os.Remove(tmp) }, nil
}

func (b *BulkImporter) downloadToTemp(ctx context.Context, urlStr, pattern string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", bduFeedUserAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := b.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(slurp)))
	}

	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

type vulXMLRecord struct {
	Identifier      string `xml:"identifier"`
	Name            string `xml:"name"`
	Description     string `xml:"description"`
	IdentifyDate    string `xml:"identify_date"`
	PublicationDate string `xml:"publication_date"`
	LastUpdDate     string `xml:"last_upd_date"`
	Severity        string `xml:"severity"`
	VulState        string `xml:"vul_state"`
	Identifiers     struct {
		Items []struct {
			Type string `xml:"type,attr"`
			Link string `xml:"link,attr"`
			Text string `xml:",chardata"`
		} `xml:"identifier"`
	} `xml:"identifiers"`
	CWEs *struct {
		List []struct {
			ID   string `xml:"identifier"`
			Name string `xml:"name"`
		} `xml:"cwe"`
	} `xml:"cwes"`
}

func (v vulXMLRecord) toSourceRecord() models.SourceRecord {
	title := strings.TrimSpace(v.Name)
	if title == "" {
		title = v.Identifier
	}
	pub := parseBDUDDMY(v.PublicationDate)
	mod := parseBDUDDMY(v.LastUpdDate)
	if mod == nil {
		mod = pub
	}

	aliases := make([]models.ReferenceAlias, 0, 8)
	seen := map[string]struct{}{}
	add := func(t, val string) {
		val = strings.TrimSpace(val)
		if val == "" {
			return
		}
		key := t + ":" + val
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		aliases = append(aliases, models.ReferenceAlias{AliasType: t, AliasValue: val})
	}

	add("BDU", v.Identifier)
	for _, id := range v.Identifiers.Items {
		t := strings.TrimSpace(id.Type)
		txt := strings.TrimSpace(id.Text)
		if t == "CVE" {
			for _, cve := range cvePattern.FindAllString(txt, -1) {
				add("CVE", cve)
			}
		} else if t != "" && txt != "" {
			add(t, txt)
		}
	}
	if v.CWEs != nil {
		for _, cwe := range v.CWEs.List {
			if id := strings.TrimSpace(cwe.ID); id != "" {
				add("CWE", id)
			}
		}
	}

	meta := map[string]any{
		"bdu_import":    "vulxml",
		"identify_date": strings.TrimSpace(v.IdentifyDate),
		"vul_state":     strings.TrimSpace(v.VulState),
	}
	raw, _ := json.Marshal(v)

	return models.SourceRecord{
		ExternalID:  strings.TrimSpace(v.Identifier),
		Title:       title,
		Description: strings.TrimSpace(v.Description),
		Severity:    strings.TrimSpace(v.Severity),
		PublishedAt: pub,
		ModifiedAt:  mod,
		SourceURL:   bduDetailURL(v.Identifier),
		Status:      strings.TrimSpace(v.VulState),
		Metadata:    mustJSON(meta),
		Aliases:     aliases,
		RawPayload:  string(raw),
		ContentType: "application/xml",
	}
}

func bduDetailURL(bduID string) string {
	const base = "https://bdu.fstec.ru/vul/bdu"
	if strings.TrimSpace(bduID) == "" {
		return base
	}
	return base + "?id=" + strings.TrimSpace(bduID)
}

func parseBDUDDMY(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{"02.01.2006", "2.1.2006"} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return &t
		}
	}
	return nil
}

func (b *BulkImporter) importVullistXLSX(ctx context.Context, path string, onBatch func([]models.SourceRecord) error) error {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		return fmt.Errorf("empty workbook")
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return err
	}

	headerRow := -1
	var headers []string
	for i, row := range rows {
		if row == nil {
			continue
		}
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), "идентификатор") {
				headerRow = i
				headers = row
				break
			}
		}
		if headerRow >= 0 {
			break
		}
	}
	if headerRow < 0 {
		return fmt.Errorf("header row with «Идентификатор» not found")
	}

	idCol := findColumn(headers, "идентификатор")
	nameCol := findColumn(headers, "наименование")
	descCol := findColumn(headers, "описание")
	if idCol < 0 {
		return fmt.Errorf("column «Идентификатор» not found")
	}

	batch := make([]models.SourceRecord, 0, b.BatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		cp := make([]models.SourceRecord, len(batch))
		copy(cp, batch)
		batch = batch[:0]
		return onBatch(cp)
	}

	for ri := headerRow + 1; ri < len(rows); ri++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		row := rows[ri]
		if len(row) <= idCol {
			continue
		}
		extID := strings.TrimSpace(row[idCol])
		if extID == "" || !strings.HasPrefix(strings.ToUpper(extID), "BDU:") {
			continue
		}

		title := extID
		if nameCol >= 0 && nameCol < len(row) {
			if t := strings.TrimSpace(row[nameCol]); t != "" {
				title = t
			}
		}
		desc := ""
		if descCol >= 0 && descCol < len(row) {
			desc = strings.TrimSpace(row[descCol])
		}

		meta := map[string]any{"bdu_import": "vullist"}
		for c, head := range headers {
			if c >= len(row) {
				continue
			}
			h := strings.TrimSpace(head)
			if h == "" {
				continue
			}
			meta["col_"+strings.ToLower(strings.ReplaceAll(h, " ", "_"))] = strings.TrimSpace(row[c])
		}
		raw, _ := json.Marshal(meta)

		batch = append(batch, models.SourceRecord{
			ExternalID:  extID,
			Title:       title,
			Description: desc,
			PublishedAt: nil,
			ModifiedAt:  nil,
			SourceURL:   bduDetailURL(extID),
			Status:      "published",
			Metadata:    raw,
			Aliases: []models.ReferenceAlias{
				{AliasType: "BDU", AliasValue: extID},
			},
			RawPayload:  string(raw),
			ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		})
		if len(batch) >= b.BatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

func findColumn(headers []string, needle string) int {
	n := strings.ToLower(needle)
	for i, h := range headers {
		if strings.Contains(strings.ToLower(strings.TrimSpace(h)), n) {
			return i
		}
	}
	return -1
}
