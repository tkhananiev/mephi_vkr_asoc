package products

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConsoleProduct struct {
	ID                      int64     `json:"id"`
	Name                    string    `json:"name"`
	Description             string    `json:"description"`
	RepositoryURL           string    `json:"repository_url"`
	RepositoryRef           string    `json:"repository_ref"`
	RepositorySubdirectory  string    `json:"repository_subdirectory"`
	ScanTargetPath          string    `json:"scan_target_path"`
	CreatedAt               time.Time `json:"created_at"`
}

type CreateInput struct {
	Name                   string
	Description            string
	RepositoryURL          string
	RepositoryRef          string
	RepositorySubdirectory string
	ScanTargetPath         string
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListByOwner(ctx context.Context, ownerID int64) ([]ConsoleProduct, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, repository_url, repository_ref, repository_subdirectory, scan_target_path, created_at
		FROM core.console_products
		WHERE owner_user_id = $1
		ORDER BY created_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ConsoleProduct
	for rows.Next() {
		var p ConsoleProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.RepositoryURL, &p.RepositoryRef, &p.RepositorySubdirectory, &p.ScanTargetPath, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Create(ctx context.Context, ownerID int64, in CreateInput) (ConsoleProduct, error) {
	ref := strings.TrimSpace(in.RepositoryRef)
	if ref == "" {
		ref = "main"
	}
	sub := strings.TrimSpace(in.RepositorySubdirectory)
	sub = strings.ReplaceAll(sub, "\\", "/")
	sub = strings.Trim(sub, "/")

	scanPath := strings.TrimSpace(in.ScanTargetPath)
	if scanPath != "" && !strings.HasSuffix(scanPath, "/") {
		scanPath += "/"
	}

	var p ConsoleProduct
	err := s.pool.QueryRow(ctx, `
		INSERT INTO core.console_products (
			owner_user_id, name, description, repository_url, repository_ref, repository_subdirectory, scan_target_path
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, description, repository_url, repository_ref, repository_subdirectory, scan_target_path, created_at
	`,
		ownerID,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Description),
		strings.TrimSpace(in.RepositoryURL),
		ref,
		sub,
		scanPath,
	).Scan(&p.ID, &p.Name, &p.Description, &p.RepositoryURL, &p.RepositoryRef, &p.RepositorySubdirectory, &p.ScanTargetPath, &p.CreatedAt)
	return p, err
}
