package products

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProductNotFound = errors.New("console product not found")

type ConsoleProduct struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	RepositoryURL          string    `json:"repository_url"`
	RepositoryRef          string    `json:"repository_ref"`
	RepositoryBranchRefs   []string  `json:"repository_branch_refs"`
	RepositorySubdirectory string    `json:"repository_subdirectory"`
	ScanTargetPath         string    `json:"scan_target_path"`
	CreatedAt              time.Time `json:"created_at"`
}

type CreateInput struct {
	Name                   string
	Description            string
	RepositoryURL          string
	RepositoryRef          string
	RepositoryBranchRefs   []string
	RepositorySubdirectory string
	ScanTargetPath         string
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func NormalizeBranchRefs(repositoryRef string, branchList []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, b := range branchList {
		t := strings.TrimSpace(b)
		if t == "" {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) > 0 {
		return out
	}
	t := strings.TrimSpace(repositoryRef)
	if t == "" {
		return []string{"main"}
	}
	return []string{t}
}

func decodeBranchRefsJSON(raw []byte, fallbackRef string) []string {
	var parsed []string
	if len(raw) > 0 && json.Unmarshal(raw, &parsed) == nil && len(parsed) > 0 {
		out := NormalizeBranchRefs("", parsed)
		if len(out) > 0 {
			return out
		}
	}
	return NormalizeBranchRefs(fallbackRef, nil)
}

func (s *Store) ListByOwner(ctx context.Context, ownerID int64) ([]ConsoleProduct, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, repository_url, repository_ref, repository_branch_refs, repository_subdirectory, scan_target_path, created_at
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
		var (
			p     ConsoleProduct
			rawBZ []byte
		)
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.RepositoryURL, &p.RepositoryRef, &rawBZ, &p.RepositorySubdirectory, &p.ScanTargetPath, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.RepositoryBranchRefs = decodeBranchRefsJSON(rawBZ, p.RepositoryRef)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ProductOwnedBy(ctx context.Context, productID, ownerUserID int64) (bool, error) {
	var one int32
	err := s.pool.QueryRow(ctx, `
		SELECT 1 FROM core.console_products WHERE id = $1 AND owner_user_id = $2 LIMIT 1
	`, productID, ownerUserID).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Create(ctx context.Context, ownerID int64, in CreateInput) (ConsoleProduct, error) {
	refs := NormalizeBranchRefs(in.RepositoryRef, in.RepositoryBranchRefs)
	ref := refs[0]
	refsJSON, err := json.Marshal(refs)
	if err != nil {
		return ConsoleProduct{}, err
	}

	sub := strings.TrimSpace(in.RepositorySubdirectory)
	sub = strings.ReplaceAll(sub, "\\", "/")
	sub = strings.Trim(sub, "/")

	scanPath := strings.TrimSpace(in.ScanTargetPath)
	if scanPath != "" && !strings.HasSuffix(scanPath, "/") {
		scanPath += "/"
	}

	var (
		p     ConsoleProduct
		rawBZ []byte
	)
	err = s.pool.QueryRow(ctx, `
		INSERT INTO core.console_products (
			owner_user_id, name, description, repository_url, repository_ref, repository_branch_refs,
			repository_subdirectory, scan_target_path
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
		RETURNING id, name, description, repository_url, repository_ref, repository_branch_refs, repository_subdirectory, scan_target_path, created_at
	`,
		ownerID,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Description),
		strings.TrimSpace(in.RepositoryURL),
		ref,
		refsJSON,
		sub,
		scanPath,
	).Scan(&p.ID, &p.Name, &p.Description, &p.RepositoryURL, &p.RepositoryRef, &rawBZ, &p.RepositorySubdirectory, &p.ScanTargetPath, &p.CreatedAt)
	if err != nil {
		return ConsoleProduct{}, err
	}
	p.RepositoryBranchRefs = decodeBranchRefsJSON(rawBZ, p.RepositoryRef)
	return p, nil
}

func (s *Store) Update(ctx context.Context, ownerID, productID int64, in CreateInput) (ConsoleProduct, error) {
	refs := NormalizeBranchRefs(in.RepositoryRef, in.RepositoryBranchRefs)
	ref := refs[0]
	refsJSON, err := json.Marshal(refs)
	if err != nil {
		return ConsoleProduct{}, err
	}

	sub := strings.TrimSpace(in.RepositorySubdirectory)
	sub = strings.ReplaceAll(sub, "\\", "/")
	sub = strings.Trim(sub, "/")

	scanPath := strings.TrimSpace(in.ScanTargetPath)
	if scanPath != "" && !strings.HasSuffix(scanPath, "/") {
		scanPath += "/"
	}

	var (
		p     ConsoleProduct
		rawBZ []byte
	)
	err = s.pool.QueryRow(ctx, `
		UPDATE core.console_products SET
			name = $3,
			description = $4,
			repository_url = $5,
			repository_ref = $6,
			repository_branch_refs = $7::jsonb,
			repository_subdirectory = $8,
			scan_target_path = $9,
			updated_at = NOW()
		WHERE id = $1 AND owner_user_id = $2
		RETURNING id, name, description, repository_url, repository_ref, repository_branch_refs, repository_subdirectory, scan_target_path, created_at
	`,
		productID,
		ownerID,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Description),
		strings.TrimSpace(in.RepositoryURL),
		ref,
		refsJSON,
		sub,
		scanPath,
	).Scan(&p.ID, &p.Name, &p.Description, &p.RepositoryURL, &p.RepositoryRef, &rawBZ, &p.RepositorySubdirectory, &p.ScanTargetPath, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConsoleProduct{}, ErrProductNotFound
	}
	if err != nil {
		return ConsoleProduct{}, err
	}
	p.RepositoryBranchRefs = decodeBranchRefsJSON(rawBZ, p.RepositoryRef)
	return p, nil
}

func (s *Store) Delete(ctx context.Context, ownerID, productID int64) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM core.console_products WHERE id = $1 AND owner_user_id = $2
	`, productID, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}
