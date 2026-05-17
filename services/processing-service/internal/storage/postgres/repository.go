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

func (r *Repository) StartRun(ctx context.Context, sourceName string, findingsReceived int, ownerUserID *int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.processing_runs (source_name, status, findings_received, owner_user_id)
		VALUES ($1, 'running', $2, $3)
		RETURNING id
	`, sourceName, findingsReceived, ownerUserID).Scan(&id)
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
		ORDER BY rr.updated_at DESC
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

func (r *Repository) ListGroups(ctx context.Context, limit int, ownerUserID *int64) ([]models.VulnerabilityGroup, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, group_key, grouping_rule, severity_max, assets_count, status
		FROM core.vulnerability_groups
		WHERE (
			$2::bigint IS NULL
			OR group_key LIKE ('u:' || $2::bigint::text || ':%')
		)
		ORDER BY updated_at DESC
		LIMIT $1
	`, limit, ownerUserID)
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

func (r *Repository) ListVulnerabilityReport(ctx context.Context, limit int, ownerUserID *int64) ([]models.VulnerabilityReportRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		WITH scanner_pick AS (
			SELECT DISTINCT ON (fv.vulnerability_id)
				fv.vulnerability_id,
				f.scanner_name,
				pr.started_at AS run_started_at
			FROM core.finding_vulnerabilities fv
			JOIN core.findings f ON f.id = fv.finding_id
			LEFT JOIN core.processing_runs pr ON pr.id = f.processing_run_id
			WHERE (
				$2::bigint IS NULL
				OR pr.owner_user_id = $2
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
			COALESCE(rr.source_code, '')
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
				WHERE fv_o.vulnerability_id = v.id AND pr_o.owner_user_id = $2
			)
		)
		ORDER BY g.id, v.id
		LIMIT $1
	`, limit, ownerUserID)
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
