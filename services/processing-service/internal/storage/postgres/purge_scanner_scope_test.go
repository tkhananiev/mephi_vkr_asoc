package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPurgeScannerScopeKeepsVulnerabilitiesLinkedOutsideScope(t *testing.T) {
	dsn := os.Getenv("PROCESSING_SERVICE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PROCESSING_SERVICE_TEST_DATABASE_URL to run postgres purge regression test")
	}
	if os.Getenv("PROCESSING_SERVICE_TEST_DATABASE_ALLOW_RESET") != "1" {
		t.Skip("set PROCESSING_SERVICE_TEST_DATABASE_ALLOW_RESET=1 to allow resetting test schemas")
	}
	if !strings.Contains(strings.ToLower(dsn), "test") {
		t.Fatalf("refusing to reset database whose DSN does not contain test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	defer pool.Close()

	resetProcessingPurgeTestSchema(t, ctx, pool)

	repo := New(pool)
	ownerA := int64(101)
	ownerB := int64(202)
	productA := int64(1001)
	productB := int64(2002)

	var vulnID int64
	if err := pool.QueryRow(ctx, `INSERT INTO core.vulnerabilities DEFAULT VALUES RETURNING id`).Scan(&vulnID); err != nil {
		t.Fatalf("insert vulnerability: %v", err)
	}
	insertFindingLink(t, ctx, pool, "semgrep", ownerA, productA, vulnID)
	insertFindingLink(t, ctx, pool, "semgrep", ownerB, productB, vulnID)

	if err := repo.PurgeScannerScope(ctx, "semgrep", &ownerA, &productA); err != nil {
		t.Fatalf("purge scanner scope: %v", err)
	}

	assertCount(t, ctx, pool, "shared vulnerability", `
		SELECT COUNT(*) FROM core.vulnerabilities WHERE id = $1
	`, 1, vulnID)
	assertCount(t, ctx, pool, "other owner finding link", `
		SELECT COUNT(*)
		FROM core.finding_vulnerabilities fv
		JOIN core.findings f ON f.id = fv.finding_id
		JOIN core.processing_runs pr ON pr.id = f.processing_run_id
		WHERE fv.vulnerability_id = $1
		  AND pr.owner_user_id = $2
		  AND pr.console_product_id = $3
	`, 1, vulnID, ownerB, productB)
	assertCount(t, ctx, pool, "purged owner run", `
		SELECT COUNT(*) FROM core.processing_runs
		WHERE owner_user_id = $1 AND console_product_id = $2
	`, 0, ownerA, productA)
}

func resetProcessingPurgeTestSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		DROP SCHEMA IF EXISTS integration CASCADE;
		DROP SCHEMA IF EXISTS core CASCADE;
		CREATE SCHEMA core;
		CREATE SCHEMA integration;

		CREATE TABLE core.processing_runs (
			id BIGSERIAL PRIMARY KEY,
			source_name TEXT NOT NULL,
			owner_user_id BIGINT,
			console_product_id BIGINT
		);
		CREATE TABLE core.findings (
			id BIGSERIAL PRIMARY KEY,
			processing_run_id BIGINT REFERENCES core.processing_runs (id) ON DELETE SET NULL
		);
		CREATE TABLE core.vulnerabilities (
			id BIGSERIAL PRIMARY KEY
		);
		CREATE TABLE core.finding_vulnerabilities (
			finding_id BIGINT NOT NULL REFERENCES core.findings (id) ON DELETE CASCADE,
			vulnerability_id BIGINT NOT NULL REFERENCES core.vulnerabilities (id) ON DELETE CASCADE,
			PRIMARY KEY (finding_id, vulnerability_id)
		);
		CREATE TABLE core.vulnerability_groups (
			id BIGSERIAL PRIMARY KEY,
			group_key TEXT NOT NULL
		);
		CREATE TABLE core.group_vulnerabilities (
			group_id BIGINT NOT NULL REFERENCES core.vulnerability_groups (id) ON DELETE CASCADE,
			vulnerability_id BIGINT NOT NULL REFERENCES core.vulnerabilities (id) ON DELETE CASCADE,
			PRIMARY KEY (group_id, vulnerability_id)
		);
		CREATE TABLE integration.ticket_links (
			group_id BIGINT PRIMARY KEY
		);
	`)
	if err != nil {
		t.Fatalf("reset test schema: %v", err)
	}
}

func insertFindingLink(t *testing.T, ctx context.Context, pool *pgxpool.Pool, scanner string, ownerID, productID, vulnID int64) {
	t.Helper()
	var runID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO core.processing_runs (source_name, owner_user_id, console_product_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, scanner, ownerID, productID).Scan(&runID); err != nil {
		t.Fatalf("insert run: %v", err)
	}
	var findingID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO core.findings (processing_run_id)
		VALUES ($1)
		RETURNING id
	`, runID).Scan(&findingID); err != nil {
		t.Fatalf("insert finding: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO core.finding_vulnerabilities (finding_id, vulnerability_id)
		VALUES ($1, $2)
	`, findingID, vulnID); err != nil {
		t.Fatalf("insert finding vulnerability link: %v", err)
	}
}

func assertCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, label string, query string, want int64, args ...interface{}) {
	t.Helper()
	var got int64
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("%s count: %v", label, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", label, got, want)
	}
}
