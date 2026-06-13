package postgres

import (
	"context"
	"fmt"
	"strings"
)

type purgeStatement struct {
	query string
	args  []any
}

// сканера в рамках owner_user_id и console_product_id (новый прогон = снимок, без накопления).
func (r *Repository) PurgeScannerScope(ctx context.Context, scannerName string, ownerUserID *int64, consoleProductID *int64) error {
	scanner := strings.TrimSpace(scannerName)
	if scanner == "" {
		return fmt.Errorf("scanner name required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, stmt := range purgeScannerScopeStatements(scanner, ownerUserID, consoleProductID) {
		if _, err := tx.Exec(ctx, stmt.query, stmt.args...); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func purgeScannerScopeStatements(scanner string, ownerUserID *int64, consoleProductID *int64) []purgeStatement {
	return []purgeStatement{
		{
			query: `
		DELETE FROM core.group_vulnerabilities gv
		USING core.vulnerability_groups g
		WHERE gv.group_id = g.id
		  AND gv.vulnerability_id IN (
			SELECT DISTINCT fv.vulnerability_id
			FROM core.finding_vulnerabilities fv
			INNER JOIN core.findings f ON f.id = fv.finding_id
			INNER JOIN core.processing_runs pr ON pr.id = f.processing_run_id
			WHERE lower(btrim(pr.source_name)) = lower(btrim($1))
			  AND ($2::bigint IS NULL OR pr.owner_user_id IS NOT DISTINCT FROM $2::bigint)
			  AND ($3::bigint IS NULL OR pr.console_product_id IS NOT DISTINCT FROM $3::bigint)
		  )
		  AND (
			$2::bigint IS NULL
			OR g.group_key LIKE ('u:' || $2::bigint::text || ':%')
		  )
	`,
			args: []any{scanner, ownerUserID, consoleProductID},
		},
		{
			query: `
		DELETE FROM core.finding_vulnerabilities fv
		USING core.findings f, core.processing_runs pr
		WHERE fv.finding_id = f.id
		  AND f.processing_run_id = pr.id
		  AND lower(btrim(pr.source_name)) = lower(btrim($1))
		  AND ($2::bigint IS NULL OR pr.owner_user_id IS NOT DISTINCT FROM $2::bigint)
		  AND ($3::bigint IS NULL OR pr.console_product_id IS NOT DISTINCT FROM $3::bigint)
	`,
			args: []any{scanner, ownerUserID, consoleProductID},
		},
		{
			query: `
		DELETE FROM core.findings f
		USING core.processing_runs pr
		WHERE f.processing_run_id = pr.id
		  AND lower(btrim(pr.source_name)) = lower(btrim($1))
		  AND ($2::bigint IS NULL OR pr.owner_user_id IS NOT DISTINCT FROM $2::bigint)
		  AND ($3::bigint IS NULL OR pr.console_product_id IS NOT DISTINCT FROM $3::bigint)
	`,
			args: []any{scanner, ownerUserID, consoleProductID},
		},
		{
			query: `
		DELETE FROM core.vulnerabilities v
		WHERE NOT EXISTS (
			SELECT 1
			FROM core.finding_vulnerabilities fv
			INNER JOIN core.findings f ON f.id = fv.finding_id
			INNER JOIN core.processing_runs pr ON pr.id = f.processing_run_id
			WHERE fv.vulnerability_id = v.id
			  AND ($1::bigint IS NULL OR pr.owner_user_id IS NOT DISTINCT FROM $1::bigint)
			  AND ($2::bigint IS NULL OR pr.console_product_id IS NOT DISTINCT FROM $2::bigint)
		)
	`,
			args: []any{ownerUserID, consoleProductID},
		},
		{
			query: `
		DELETE FROM integration.ticket_links tl
		USING core.vulnerability_groups g
		WHERE tl.group_id = g.id
		  AND NOT EXISTS (
			SELECT 1 FROM core.group_vulnerabilities gv WHERE gv.group_id = g.id
		  )
		  AND (
			$1::bigint IS NULL
			OR g.group_key LIKE ('u:' || $1::bigint::text || ':%')
		  )
	`,
			args: []any{ownerUserID},
		},
		{
			query: `
		DELETE FROM core.vulnerability_groups g
		WHERE NOT EXISTS (
			SELECT 1 FROM core.group_vulnerabilities gv WHERE gv.group_id = g.id
		)
		  AND (
			$1::bigint IS NULL
			OR g.group_key LIKE ('u:' || $1::bigint::text || ':%')
		  )
	`,
			args: []any{ownerUserID},
		},
		{
			query: `
		DELETE FROM core.processing_runs pr
		WHERE lower(btrim(pr.source_name)) = lower(btrim($1))
		  AND ($2::bigint IS NULL OR pr.owner_user_id IS NOT DISTINCT FROM $2::bigint)
		  AND ($3::bigint IS NULL OR pr.console_product_id IS NOT DISTINCT FROM $3::bigint)
	`,
			args: []any{scanner, ownerUserID, consoleProductID},
		},
	}
}
