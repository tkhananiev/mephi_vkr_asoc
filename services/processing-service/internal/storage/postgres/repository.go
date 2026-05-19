package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"mephi_vkr_asoc/services/processing-service/internal/models"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) StartRun(ctx context.Context, sourceName string, findingsReceived int, ownerUserID *int64, channel string, consoleProductID *int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.processing_runs (source_name, status, findings_received, owner_user_id, channel, console_product_id)
		VALUES ($1, 'running', $2, $3, $4, $5)
		RETURNING id
	`, sourceName, findingsReceived, ownerUserID, channel, consoleProductID).Scan(&id)
	return id, err
}

func (r *Repository) FinishRun(ctx context.Context, runID int64, status string, result models.ProcessingResult, errMsg *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.processing_runs
		SET status = $2,
		    finished_at = NOW(),
		    findings_processed = $3,
		    vulnerabilities_created = $4,
		    groups_updated = $5,
		    error_message = $6
		WHERE id = $1
	`, runID, status, result.FindingsProcessed, result.VulnerabilitiesCreated, result.GroupsUpdated, errMsg)
	return err
}

func (r *Repository) InsertFinding(ctx context.Context, finding models.Finding) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.findings (
			processing_run_id, scanner_name, asset_id, raw_identifier, normalized_identifier,
			severity, component, version, payload_json
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id
	`, finding.ProcessingRunID, finding.ScannerName, finding.AssetID, finding.RawIdentifier,
		finding.NormalizedIdentifier, finding.Severity, finding.Component, finding.Version, finding.PayloadJSON).Scan(&id)
	return id, err
}

func (r *Repository) FindReferenceRecordIDByCVE(ctx context.Context, cve string) (*int64, error) {
	if strings.TrimSpace(cve) == "" {
		return nil, nil
	}
	var id int64
	err := r.pool.QueryRow(ctx, `
		SELECT rr.id
		FROM catalog.reference_records rr
		JOIN catalog.reference_aliases ra ON ra.reference_record_id = rr.id
		WHERE ra.alias_type = 'CVE' AND ra.alias_value = $1
		ORDER BY rr.updated_at DESC
		LIMIT 1
	`, cve).Scan(&id)
	if err != nil {
		return nil, nil
	}
	return &id, nil
}

func (r *Repository) FindReferenceRecordIDByCWE(ctx context.Context, cwe string) (*int64, error) {
	cwe = normalizeCWEAliasValue(cwe)
	if cwe == "" {
		return nil, nil
	}
	var id int64
	err := r.pool.QueryRow(ctx, `
		SELECT rr.id
		FROM catalog.reference_records rr
		JOIN catalog.reference_aliases ra ON ra.reference_record_id = rr.id
		WHERE ra.alias_type = 'CWE' AND ra.alias_value = $1
		ORDER BY
			CASE WHEN rr.source_code = 'nvd' THEN 0 ELSE 1 END,
			rr.published_at DESC NULLS LAST,
			rr.updated_at DESC,
			rr.external_id DESC
		LIMIT 1
	`, cwe).Scan(&id)
	if err != nil {
		return nil, nil
	}
	return &id, nil
}

func (r *Repository) FindCVEAliasByReferenceRecordID(ctx context.Context, referenceRecordID int64) (string, error) {
	var alias string
	err := r.pool.QueryRow(ctx, `
		SELECT alias_value
		FROM catalog.reference_aliases
		WHERE reference_record_id = $1 AND alias_type = 'CVE'
		ORDER BY alias_value
		LIMIT 1
	`, referenceRecordID).Scan(&alias)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(alias), nil
}

// normalizeCWEAliasValue приводит к виду CWE-<id> для сопоставления с catalog.reference_aliases.
func normalizeCWEAliasValue(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u := strings.ToUpper(s)
	if strings.HasPrefix(u, "CWE-") {
		return u
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return "CWE-" + u
}

func (r *Repository) CreateVulnerability(ctx context.Context, vulnerability models.Vulnerability) (int64, bool, error) {
	var (
		id int64
	)
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.vulnerabilities (
			cve_id, product, version, cwe, normalized_severity, correlation_status, reference_record_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT DO NOTHING
		RETURNING id
	`, vulnerability.CVEID, vulnerability.Product, vulnerability.Version, vulnerability.CWE,
		vulnerability.NormalizedSeverity, vulnerability.CorrelationStatus, vulnerability.ReferenceRecordID).Scan(&id)
	if err == nil {
		return id, true, nil
	}

	err = r.pool.QueryRow(ctx, `
		SELECT id
		FROM core.vulnerabilities
		WHERE COALESCE(cve_id, '') = COALESCE($1, '')
		  AND COALESCE(product, '') = COALESCE($2, '')
		  AND COALESCE(version, '') = COALESCE($3, '')
		  AND COALESCE(cwe, '') = COALESCE($4, '')
		ORDER BY id
		LIMIT 1
	`, vulnerability.CVEID, vulnerability.Product, vulnerability.Version, vulnerability.CWE).Scan(&id)
	if err != nil {
		return 0, false, err
	}
	return id, false, nil
}

// MergeVulnerabilityCatalog — при повторных прогонах не создаём дубликаты по сигнатуре (CreateVulnerability), но может
// «дорос» каталог (CWE aliased в NVD). Тогда дорисовываем cve_id / reference_record_id у существующей строки.
func (r *Repository) MergeVulnerabilityCatalog(ctx context.Context, vulnerabilityID int64, cve string, referenceRecordID *int64, correlationStatus string) error {
	var ref pgtype.Int8
	if referenceRecordID != nil {
		ref.Valid = true
		ref.Int64 = *referenceRecordID
	}
	cveTrim := strings.TrimSpace(cve)
	status := strings.TrimSpace(correlationStatus)
	if status == "" {
		status = "not_found"
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE core.vulnerabilities v
		SET
			cve_id = CASE
				WHEN COALESCE(TRIM(v.cve_id), '') = '' AND LENGTH($2::text) > 0
				THEN $2::text
				ELSE v.cve_id
			END,
			reference_record_id = COALESCE(v.reference_record_id, $3),
			correlation_status = CASE
				WHEN v.correlation_status IN ('not_found', 'pending') AND $4::text IS NOT NULL AND $4::text <> '' AND $4::text <> 'not_found'
				THEN $4::text
				ELSE v.correlation_status
			END,
			updated_at = NOW()
		WHERE v.id = $1
		  AND (
				(COALESCE(TRIM(v.cve_id), '') = '' AND LENGTH($2::text) > 0)
			 OR (v.reference_record_id IS NULL AND $3::bigint IS NOT NULL)
			 OR (v.correlation_status IN ('not_found', 'pending') AND COALESCE(NULLIF(trim($4::text), ''), '') NOT IN ('', 'not_found'))
		  )
	`, vulnerabilityID, cveTrim, ref, status)
	return err
}

func (r *Repository) LinkFindingToVulnerability(ctx context.Context, findingID, vulnerabilityID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO core.finding_vulnerabilities (finding_id, vulnerability_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, findingID, vulnerabilityID)
	return err
}

func (r *Repository) UpsertGroup(ctx context.Context, groupKey, severity, groupingRule string) (int64, bool, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.vulnerability_groups (group_key, grouping_rule, severity_max, assets_count, status)
		VALUES ($1, $2, $3, 1, 'open')
		ON CONFLICT (group_key)
		DO UPDATE SET
			severity_max = EXCLUDED.severity_max,
			assets_count = core.vulnerability_groups.assets_count + 1,
			updated_at = NOW()
		RETURNING id
	`, groupKey, groupingRule, severity).Scan(&id)
	return id, true, err
}

func (r *Repository) LinkGroupToVulnerability(ctx context.Context, groupID, vulnerabilityID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO core.group_vulnerabilities (group_id, vulnerability_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, groupID, vulnerabilityID)
	return err
}

func (r *Repository) ListGroups(ctx context.Context, limit int, ownerUserID *int64, consoleProductID *int64) ([]models.VulnerabilityGroup, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.group_key, g.grouping_rule, g.severity_max, g.assets_count, g.status
		FROM core.vulnerability_groups g
		WHERE (
			$2::bigint IS NULL
			OR g.group_key LIKE ('u:' || $2::bigint::text || ':%')
		)
		AND (
			$3::bigint IS NULL
			OR EXISTS (
				SELECT 1
				FROM core.group_vulnerabilities gv
				JOIN core.vulnerabilities v ON v.id = gv.vulnerability_id
				JOIN core.finding_vulnerabilities fv ON fv.vulnerability_id = v.id
				JOIN core.findings f ON f.id = fv.finding_id
				JOIN core.processing_runs pr ON pr.id = f.processing_run_id
				WHERE gv.group_id = g.id
				  AND ($2::bigint IS NULL OR pr.owner_user_id = $2)
				  AND pr.console_product_id = $3
			)
		)
		ORDER BY g.updated_at DESC
		LIMIT $1
	`, limit, ownerUserID, consoleProductID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.VulnerabilityGroup, 0, limit)
	for rows.Next() {
		var item models.VulnerabilityGroup
		if err := rows.Scan(&item.ID, &item.GroupKey, &item.GroupingRule, &item.SeverityMax, &item.AssetsCount, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func reportFilterArg(f *models.VulnerabilityReportFilter, pick func(*models.VulnerabilityReportFilter) *string) interface{} {
	if f == nil {
		return nil
	}
	p := pick(f)
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return s
}

func (r *Repository) ListVulnerabilityReport(ctx context.Context, limit int, ownerUserID *int64, consoleProductID *int64, filter *models.VulnerabilityReportFilter) ([]models.VulnerabilityReportRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	fk := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.GroupKey })
	fcve := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.CVE })
	fbdu := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.BDUID })
	fscan := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.ScannerName })
	fsev := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.Severity })
	fcat := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.CatalogSource })
	frun := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.RunChannel })
	fass := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.AssetPath })
	fver := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.Version })
	fcwe := reportFilterArg(filter, func(fr *models.VulnerabilityReportFilter) *string { return fr.CWE })

	var fgidArg, fvidArg interface{}
	var runFromArg, runToArg interface{}
	if filter != nil {
		if filter.GroupID != nil {
			fgidArg = *filter.GroupID
		}
		if filter.VulnerabilityID != nil {
			fvidArg = *filter.VulnerabilityID
		}
		if filter.RunAtFrom != nil {
			runFromArg = *filter.RunAtFrom
		}
		if filter.RunAtTo != nil {
			runToArg = *filter.RunAtTo
		}
	}

	rows, err := r.pool.Query(ctx, `
		WITH scanner_pick AS (
			SELECT DISTINCT ON (fv.vulnerability_id)
				fv.vulnerability_id,
				f.scanner_name,
				pr.started_at AS run_started_at,
				COALESCE(NULLIF(trim(pr.channel), ''), 'manual') AS run_channel
			FROM core.finding_vulnerabilities fv
			JOIN core.findings f ON f.id = fv.finding_id
			LEFT JOIN core.processing_runs pr ON pr.id = f.processing_run_id
			WHERE (
				$2::bigint IS NULL
				OR pr.owner_user_id = $2
			)
			AND (
				$3::bigint IS NULL
				OR pr.console_product_id = $3
			)
			AND (
				$7::text IS NULL OR btrim($7::text) = ''
				OR strpos(lower(f.scanner_name), lower(btrim($7::text))) > 0
			)
			ORDER BY fv.vulnerability_id, pr.started_at DESC NULLS LAST, f.id DESC
		)
		SELECT
			g.id,
			g.group_key,
			v.id,
			COALESCE(v.cve_id, ''),
			COALESCE(
				CASE WHEN rr.source_code IS NOT NULL AND rr.source_code LIKE 'bdu%' THEN rr.external_id END,
				(SELECT bdu_rr.external_id
				 FROM catalog.reference_records bdu_rr
				 INNER JOIN catalog.reference_aliases bdu_ra ON bdu_ra.reference_record_id = bdu_rr.id
				 WHERE bdu_rr.source_code LIKE 'bdu%'
				   AND TRIM(COALESCE(v.cve_id, '')) <> ''
				   AND bdu_ra.alias_type = 'CVE'
				   AND bdu_ra.alias_value = TRIM(v.cve_id)
				 ORDER BY bdu_rr.updated_at DESC
				 LIMIT 1),
				''
			),
			COALESCE(sp.scanner_name, ''),
			COALESCE(v.product, ''),
			COALESCE(v.version, ''),
			COALESCE(v.normalized_severity, ''),
			sp.run_started_at,
			COALESCE(rr.source_code, ''),
			COALESCE(sp.run_channel, 'manual')
		FROM core.vulnerabilities v
		JOIN core.group_vulnerabilities gv ON gv.vulnerability_id = v.id
		JOIN core.vulnerability_groups g ON g.id = gv.group_id
		LEFT JOIN catalog.reference_records rr ON rr.id = v.reference_record_id
		LEFT JOIN scanner_pick sp ON sp.vulnerability_id = v.id
		WHERE (
			$2::bigint IS NULL
			OR EXISTS (
				SELECT 1
				FROM core.finding_vulnerabilities fv_o
				JOIN core.findings f_o ON f_o.id = fv_o.finding_id
				LEFT JOIN core.processing_runs pr_o ON pr_o.id = f_o.processing_run_id
				WHERE fv_o.vulnerability_id = v.id
				  AND pr_o.owner_user_id = $2
				  AND (
					$3::bigint IS NULL
					OR pr_o.console_product_id = $3
				  )
			)
		)
		AND (
			$4::text IS NULL OR btrim($4::text) = ''
			OR strpos(lower(g.group_key), lower(btrim($4::text))) > 0
		)
		AND (
			$5::text IS NULL OR btrim($5::text) = ''
			OR strpos(lower(COALESCE(v.cve_id, '')), lower(btrim($5::text))) > 0
		)
		AND (
			$6::text IS NULL OR btrim($6::text) = ''
			OR strpos(lower(COALESCE(
				CASE WHEN rr.source_code IS NOT NULL AND rr.source_code LIKE 'bdu%' THEN rr.external_id END,
				(SELECT bdu_rr.external_id
				 FROM catalog.reference_records bdu_rr
				 INNER JOIN catalog.reference_aliases bdu_ra ON bdu_ra.reference_record_id = bdu_rr.id
				 WHERE bdu_rr.source_code LIKE 'bdu%'
				   AND TRIM(COALESCE(v.cve_id, '')) <> ''
				   AND bdu_ra.alias_type = 'CVE'
				   AND bdu_ra.alias_value = TRIM(v.cve_id)
				 ORDER BY bdu_rr.updated_at DESC
				 LIMIT 1),
				''
			)), lower(btrim($6::text))) > 0
		)
		AND (
			$7::text IS NULL OR btrim($7::text) = ''
			OR sp.vulnerability_id IS NOT NULL
		)
		AND (
			$8::text IS NULL OR btrim($8::text) = ''
			OR strpos(lower(COALESCE(v.normalized_severity, '')), lower(btrim($8::text))) > 0
		)
		AND (
			$9::text IS NULL OR btrim($9::text) = ''
			OR strpos(lower(COALESCE(rr.source_code, '')), lower(btrim($9::text))) > 0
		)
		AND (
			$10::text IS NULL OR btrim($10::text) = ''
			OR lower(btrim($10::text)) = lower(trim(COALESCE(
				NULLIF(trim(sp.run_channel), ''),
				'manual'
			)))
		)
		AND (
			$11::text IS NULL OR btrim($11::text) = ''
			OR strpos(lower(COALESCE(v.product, '')), lower(btrim($11::text))) > 0
		)
		AND (
			$12::text IS NULL OR btrim($12::text) = ''
			OR strpos(lower(COALESCE(v.version, '')), lower(btrim($12::text))) > 0
		)
		AND (
			$13::bigint IS NULL OR g.id = $13
		)
		AND (
			$14::bigint IS NULL OR v.id = $14
		)
		AND (
			$15::timestamptz IS NULL OR (sp.run_started_at IS NOT NULL AND sp.run_started_at >= $15)
		)
		AND (
			$16::timestamptz IS NULL OR (sp.run_started_at IS NOT NULL AND sp.run_started_at <= $16)
		)
		AND (
			$17::text IS NULL OR btrim($17::text) = ''
			OR strpos(lower(COALESCE(v.cwe, '')), lower(btrim($17::text))) > 0
		)
		ORDER BY g.id, v.id
		LIMIT $1
	`, limit, ownerUserID, consoleProductID, fk, fcve, fbdu, fscan, fsev, fcat, frun, fass, fver, fgidArg, fvidArg, runFromArg, runToArg, fcwe)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.VulnerabilityReportRow, 0, min(limit, 64))
	for rows.Next() {
		var row models.VulnerabilityReportRow
		var ts pgtype.Timestamptz
		if err := rows.Scan(
			&row.GroupID,
			&row.GroupKey,
			&row.VulnerabilityID,
			&row.CVE,
			&row.BDUID,
			&row.ScannerName,
			&row.AssetPath,
			&row.Version,
			&row.Severity,
			&ts,
			&row.CatalogSource,
			&row.RunChannel,
		); err != nil {
			return nil, err
		}
		if ts.Valid {
			t := ts.Time.In(time.UTC)
			row.RunAt = &t
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
