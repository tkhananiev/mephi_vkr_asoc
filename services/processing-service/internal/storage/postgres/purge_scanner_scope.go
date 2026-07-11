package postgres

import (
	"context"
	"fmt"
	"strings"
)

const (
	deleteScopedFindingVulnerabilitiesSQL = `
		DELETE FROM core.finding_vulnerabilities fv
		USING core.findings f, core.processing_runs pr
		WHERE fv.finding_id = f.id
		  AND f.processing_run_id = pr.id
		  AND lower(btrim(pr.source_name)) = lower(btrim($1))
		  AND pr.owner_user_id IS NOT DISTINCT FROM $2::bigint
		  AND pr.console_product_id IS NOT DISTINCT FROM $3::bigint
	`

	deleteScopedFindingsSQL = `
		DELETE FROM core.findings f
		USING core.processing_runs pr
		WHERE f.processing_run_id = pr.id
		  AND lower(btrim(pr.source_name)) = lower(btrim($1))
		  AND pr.owner_user_id IS NOT DISTINCT FROM $2::bigint
		  AND pr.console_product_id IS NOT DISTINCT FROM $3::bigint
	`

	deleteOrphanVulnerabilitiesSQL = `
		DELETE FROM core.vulnerabilities v
		WHERE NOT EXISTS (
			SELECT 1
			FROM core.finding_vulnerabilities fv
			WHERE fv.vulnerability_id = v.id
		)
	`

	deleteEmptyTicketLinksSQL = `
		DELETE FROM integration.ticket_links tl
		USING core.vulnerability_groups g
		WHERE tl.group_id = g.id
		  AND NOT EXISTS (
			SELECT 1 FROM core.group_vulnerabilities gv WHERE gv.group_id = g.id
		  )
		  AND (
			($1::bigint IS NULL AND g.group_key !~ '^u:[0-9]+:')
			OR ($1::bigint IS NOT NULL AND g.group_key LIKE ('u:' || $1::bigint::text || ':%'))
		  )
	`

	deleteEmptyGroupsSQL = `
		DELETE FROM core.vulnerability_groups g
		WHERE NOT EXISTS (
			SELECT 1 FROM core.group_vulnerabilities gv WHERE gv.group_id = g.id
		)
		  AND (
			($1::bigint IS NULL AND g.group_key !~ '^u:[0-9]+:')
			OR ($1::bigint IS NOT NULL AND g.group_key LIKE ('u:' || $1::bigint::text || ':%'))
		  )
	`

	deleteScopedProcessingRunsSQL = `
		DELETE FROM core.processing_runs pr
		WHERE lower(btrim(pr.source_name)) = lower(btrim($1))
		  AND pr.owner_user_id IS NOT DISTINCT FROM $2::bigint
		  AND pr.console_product_id IS NOT DISTINCT FROM $3::bigint
	`
)

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

	_, err = tx.Exec(ctx, deleteScopedFindingVulnerabilitiesSQL, scanner, ownerUserID, consoleProductID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, deleteScopedFindingsSQL, scanner, ownerUserID, consoleProductID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, deleteOrphanVulnerabilitiesSQL)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, deleteEmptyTicketLinksSQL, ownerUserID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, deleteEmptyGroupsSQL, ownerUserID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, deleteScopedProcessingRunsSQL, scanner, ownerUserID, consoleProductID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
