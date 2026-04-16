package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mephi_vkr_aspm/services/reference-data-service/internal/models"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) StartSyncRun(ctx context.Context, sourceCode string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO audit.reference_sync_runs (source_code, status) VALUES ($1, 'running') RETURNING id`,
		sourceCode,
	).Scan(&id)
	return id, err
}

func (r *Repository) FinishSyncRun(ctx context.Context, runID int64, status string, result models.SyncResult, errMsg *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE audit.reference_sync_runs
		SET status = $2,
		    finished_at = NOW(),
		    items_discovered = $3,
		    items_processed = $4,
		    items_inserted = $5,
		    items_updated = $6,
		    error_message = $7
		WHERE id = $1
	`, runID, status, result.ItemsDiscovered, result.ItemsProcessed, result.ItemsInserted, result.ItemsUpdated, errMsg)
	return err
}

func (r *Repository) UpsertRawItem(ctx context.Context, item models.RawItem) error {
	hash := sha256.Sum256([]byte(item.RawPayload))
	item.RawHash = hex.EncodeToString(hash[:])

	_, err := r.pool.Exec(ctx, `
		INSERT INTO raw.reference_raw_items (
			source_code, external_id, source_url, content_type, raw_payload, raw_hash
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (source_code, external_id, raw_hash) DO NOTHING
	`, item.SourceCode, item.ExternalID, item.SourceURL, item.ContentType, item.RawPayload, item.RawHash)
	return err
}

func (r *Repository) UpsertReferenceRecord(ctx context.Context, record models.ReferenceRecord) (bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var (
		recordID int64
		inserted bool
	)

	query := `
		INSERT INTO catalog.reference_records (
			source_code, external_id, title, description, severity,
			published_at, modified_at, source_url, status, metadata_json
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (source_code, external_id)
		DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			severity = EXCLUDED.severity,
			published_at = EXCLUDED.published_at,
			modified_at = EXCLUDED.modified_at,
			source_url = EXCLUDED.source_url,
			status = EXCLUDED.status,
			metadata_json = EXCLUDED.metadata_json,
			updated_at = NOW()
		RETURNING id, (xmax = 0) AS inserted
	`
	if err := tx.QueryRow(ctx, query,
		record.SourceCode,
		record.ExternalID,
		record.Title,
		record.Description,
		record.Severity,
		record.PublishedAt,
		record.ModifiedAt,
		record.SourceURL,
		record.Status,
		record.Metadata,
	).Scan(&recordID, &inserted); err != nil {
		return false, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM catalog.reference_aliases WHERE reference_record_id = $1`, recordID); err != nil {
		return false, err
	}

	for _, alias := range record.Aliases {
		if _, err := tx.Exec(ctx, `
			INSERT INTO catalog.reference_aliases (reference_record_id, alias_type, alias_value)
			VALUES ($1, $2, $3)
			ON CONFLICT (reference_record_id, alias_type, alias_value) DO NOTHING
		`, recordID, alias.AliasType, alias.AliasValue); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return inserted, nil
}

func (r *Repository) ListSyncRuns(ctx context.Context, limit int) ([]models.SyncRun, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, source_code, status, started_at, finished_at,
		       items_discovered, items_processed, items_inserted, items_updated, error_message
		FROM audit.reference_sync_runs
		ORDER BY started_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.SyncRun, 0, limit)
	for rows.Next() {
		var run models.SyncRun
		if err := rows.Scan(
			&run.ID,
			&run.SourceCode,
			&run.Status,
			&run.StartedAt,
			&run.FinishedAt,
			&run.ItemsDiscovered,
			&run.ItemsProcessed,
			&run.ItemsInserted,
			&run.ItemsUpdated,
			&run.ErrorMessage,
		); err != nil {
			return nil, err
		}
		result = append(result, run)
	}

	return result, rows.Err()
}

func MustJSON(raw any) []byte {
	data, err := json.Marshal(raw)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func WrapError(source string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", source, err)
}
