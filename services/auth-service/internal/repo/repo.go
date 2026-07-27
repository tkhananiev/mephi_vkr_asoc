package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// ErrLastEnabledAdmin is returned when delete/demote/disable would leave zero enabled admins.
var ErrLastEnabledAdmin = errors.New("cannot remove the last enabled administrator")

type ConsoleUser struct {
	ID           int64
	Email        string
	Username     string
	FirstName    string
	LastName     string
	Patronymic   string
	DisplayName  string
	PasswordHash string
	Disabled     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ConsoleUserInsert struct {
	Email         string
	Username      string
	FirstName     string
	LastName      string
	Patronymic    string
	LegacyDisplay string // если ФИО не переданы (старые клиенты)
	PasswordHash  string
}

func FormatDisplayFromFIO(last, first, patronymic string) string {
	var parts []string
	last, first, patronymic = strings.TrimSpace(last), strings.TrimSpace(first), strings.TrimSpace(patronymic)
	if last != "" {
		parts = append(parts, last)
	}
	if first != "" {
		parts = append(parts, first)
	}
	if patronymic != "" {
		parts = append(parts, patronymic)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

type Admin struct {
	ID           int64
	Login        string
	PasswordHash string
	Disabled     bool
	CreatedAt    time.Time
}

type PendingRegistration struct {
	Email          string
	Username       string
	FirstName      string
	LastName       string
	Patronymic     string
	PasswordHash   string
	VerificationCode string
	ExpiresAt      time.Time
}

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func normEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}

func (r *Repo) GetConsoleUserByLogin(ctx context.Context, identifier string) (*ConsoleUser, error) {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil, ErrNotFound
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, username, first_name, last_name, patronymic, display_name,
			password_hash, disabled, created_at, updated_at
		FROM authn.console_users
		WHERE lower(trim(email)) = lower(trim($1))
		   OR (length(trim(username)) > 0 AND lower(trim(username)) = lower(trim($1)))
	`, id)
	return scanConsoleUser(row)
}

func (r *Repo) GetConsoleUser(ctx context.Context, id int64) (*ConsoleUser, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, username, first_name, last_name, patronymic, display_name,
			password_hash, disabled, created_at, updated_at
		FROM authn.console_users WHERE id = $1
	`, id)
	return scanConsoleUser(row)
}

func scanConsoleUser(row pgx.Row) (*ConsoleUser, error) {
	var u ConsoleUser
	err := row.Scan(
		&u.ID, &u.Email, &u.Username, &u.FirstName, &u.LastName, &u.Patronymic, &u.DisplayName,
		&u.PasswordHash, &u.Disabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repo) ListConsoleUsers(ctx context.Context, limit int) ([]ConsoleUser, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, username, first_name, last_name, patronymic, display_name,
			password_hash, disabled, created_at, updated_at
		FROM authn.console_users ORDER BY id DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ConsoleUser
	for rows.Next() {
		var u ConsoleUser
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.FirstName, &u.LastName, &u.Patronymic, &u.DisplayName,
			&u.PasswordHash, &u.Disabled, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repo) InsertConsoleUser(ctx context.Context, in ConsoleUserInsert) (int64, error) {
	disp := FormatDisplayFromFIO(in.LastName, in.FirstName, in.Patronymic)
	if disp == "" {
		disp = strings.TrimSpace(in.LegacyDisplay)
	}
	un := strings.TrimSpace(in.Username)
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO authn.console_users (
			email, password_hash, display_name,
			first_name, last_name, patronymic, username
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		normEmail(in.Email),
		in.PasswordHash,
		strings.TrimSpace(disp),
		strings.TrimSpace(in.FirstName),
		strings.TrimSpace(in.LastName),
		strings.TrimSpace(in.Patronymic),
		un,
	).Scan(&id)
	return id, err
}

func (r *Repo) UpdateConsoleUser(ctx context.Context, id int64, email *string, disabled *bool, displayName *string) error {
	var sets []string
	args := []any{}
	n := 1
	if email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", n))
		args = append(args, normEmail(*email))
		n++
	}
	if disabled != nil {
		sets = append(sets, fmt.Sprintf("disabled = $%d", n))
		args = append(args, *disabled)
		n++
	}
	if displayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", n))
		args = append(args, strings.TrimSpace(*displayName))
		n++
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	q := fmt.Sprintf(`
		UPDATE authn.console_users SET %s, updated_at = NOW() WHERE id = $%d`,
		strings.Join(sets, ", "), n)
	_, err := r.pool.Exec(ctx, q, args...)
	return err
}

func (r *Repo) SetConsoleUserPassword(ctx context.Context, id int64, hash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE authn.console_users SET password_hash = $1, updated_at = NOW() WHERE id = $2
	`, hash, id)
	return err
}

func (r *Repo) AdminCount(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM authn.admins`).Scan(&n)
	return n, err
}

// CountEnabledAdmins counts administrators that can still log in (disabled = false).
func (r *Repo) CountEnabledAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM authn.admins WHERE disabled = FALSE`).Scan(&n)
	return n, err
}

// lockAdminsForUpdate serializes last-enabled checks across delete/demote/disable.
func lockAdminsForUpdate(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(ctx, `SELECT id FROM authn.admins FOR UPDATE`)
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

func countEnabledAdminsTx(ctx context.Context, tx pgx.Tx) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM authn.admins WHERE disabled = FALSE`).Scan(&n)
	return n, err
}

func (r *Repo) InsertAdmin(ctx context.Context, login, hash string) (int64, error) {
	login = strings.TrimSpace(login)
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO authn.admins (login, password_hash) VALUES ($1, $2) RETURNING id
	`, login, hash).Scan(&id)
	return id, err
}

func (r *Repo) GetAdminByLogin(ctx context.Context, login string) (*Admin, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, login, password_hash, disabled, created_at FROM authn.admins WHERE lower(login) = lower($1)
	`, strings.TrimSpace(login))
	var a Admin
	err := row.Scan(&a.ID, &a.Login, &a.PasswordHash, &a.Disabled, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repo) ListAdmins(ctx context.Context, limit int) ([]Admin, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, login, password_hash, disabled, created_at
		FROM authn.admins ORDER BY id DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Admin
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.ID, &a.Login, &a.PasswordHash, &a.Disabled, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateAdmin(ctx context.Context, id int64, login *string, disabled *bool) error {
	var sets []string
	args := []any{}
	n := 1
	if login != nil {
		sets = append(sets, fmt.Sprintf("login = $%d", n))
		args = append(args, strings.TrimSpace(*login))
		n++
	}
	if disabled != nil {
		sets = append(sets, fmt.Sprintf("disabled = $%d", n))
		args = append(args, *disabled)
		n++
	}
	if len(sets) == 0 {
		return nil
	}

	disabling := disabled != nil && *disabled
	if !disabling {
		args = append(args, id)
		q := fmt.Sprintf(`UPDATE authn.admins SET %s WHERE id = $%d`, strings.Join(sets, ", "), n)
		_, err := r.pool.Exec(ctx, q, args...)
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockAdminsForUpdate(ctx, tx); err != nil {
		return err
	}
	var currentlyDisabled bool
	err = tx.QueryRow(ctx, `SELECT disabled FROM authn.admins WHERE id = $1`, id).Scan(&currentlyDisabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !currentlyDisabled {
		enabled, err := countEnabledAdminsTx(ctx, tx)
		if err != nil {
			return err
		}
		if enabled <= 1 {
			return ErrLastEnabledAdmin
		}
	}
	args = append(args, id)
	q := fmt.Sprintf(`UPDATE authn.admins SET %s WHERE id = $%d`, strings.Join(sets, ", "), n)
	if _, err := tx.Exec(ctx, q, args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) SetAdminPassword(ctx context.Context, id int64, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE authn.admins SET password_hash = $1 WHERE id = $2`, hash, id)
	return err
}

func (r *Repo) UpsertPendingRegistration(ctx context.Context, p PendingRegistration) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO authn.pending_registrations (
			email, username, first_name, last_name, patronymic, password_hash, verification_code, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO UPDATE
		SET username = EXCLUDED.username,
		    first_name = EXCLUDED.first_name,
		    last_name = EXCLUDED.last_name,
		    patronymic = EXCLUDED.patronymic,
		    password_hash = EXCLUDED.password_hash,
		    verification_code = EXCLUDED.verification_code,
		    expires_at = EXCLUDED.expires_at
	`,
		normEmail(p.Email),
		strings.TrimSpace(p.Username),
		strings.TrimSpace(p.FirstName),
		strings.TrimSpace(p.LastName),
		strings.TrimSpace(p.Patronymic),
		p.PasswordHash,
		strings.TrimSpace(p.VerificationCode),
		p.ExpiresAt,
	)
	return err
}

func (r *Repo) GetPendingRegistration(ctx context.Context, email string) (*PendingRegistration, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT email, username, first_name, last_name, patronymic, password_hash, verification_code, expires_at
		FROM authn.pending_registrations
		WHERE lower(trim(email)) = $1
	`, normEmail(email))
	var p PendingRegistration
	err := row.Scan(
		&p.Email, &p.Username, &p.FirstName, &p.LastName, &p.Patronymic, &p.PasswordHash, &p.VerificationCode, &p.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repo) DeletePendingRegistration(ctx context.Context, email string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM authn.pending_registrations WHERE lower(trim(email)) = $1
	`, normEmail(email))
	return err
}

func (r *Repo) DeleteConsoleUser(ctx context.Context, id int64) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM authn.console_users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) DeleteAdmin(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockAdminsForUpdate(ctx, tx); err != nil {
		return err
	}
	var disabled bool
	err = tx.QueryRow(ctx, `SELECT disabled FROM authn.admins WHERE id = $1`, id).Scan(&disabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !disabled {
		enabled, err := countEnabledAdminsTx(ctx, tx)
		if err != nil {
			return err
		}
		if enabled <= 1 {
			return ErrLastEnabledAdmin
		}
	}
	cmd, err := tx.Exec(ctx, `DELETE FROM authn.admins WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *Repo) PromoteConsoleUserToAdmin(ctx context.Context, consoleUserID int64, login string) (adminID int64, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var hash string
	err = tx.QueryRow(ctx, `SELECT password_hash FROM authn.console_users WHERE id = $1`, consoleUserID).Scan(&hash)
	if err != nil {
		return 0, err
	}
	login = strings.TrimSpace(login)
	err = tx.QueryRow(ctx,
		`INSERT INTO authn.admins (login, password_hash) VALUES ($1, $2) RETURNING id`,
		login, hash).Scan(&adminID)
	if err != nil {
		return 0, err
	}
	cmd, err := tx.Exec(ctx, `DELETE FROM authn.console_users WHERE id = $1`, consoleUserID)
	if err != nil {
		return 0, err
	}
	if cmd.RowsAffected() == 0 {
		return 0, ErrNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return adminID, nil
}

func (r *Repo) DemoteAdminToConsoleUser(ctx context.Context, adminID int64, email, username, firstName, lastName, patronymic string) (consoleUserID int64, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if err := lockAdminsForUpdate(ctx, tx); err != nil {
		return 0, err
	}
	var hash string
	var disabled bool
	err = tx.QueryRow(ctx, `SELECT password_hash, disabled FROM authn.admins WHERE id = $1`, adminID).Scan(&hash, &disabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if !disabled {
		enabled, err := countEnabledAdminsTx(ctx, tx)
		if err != nil {
			return 0, err
		}
		if enabled <= 1 {
			return 0, ErrLastEnabledAdmin
		}
	}
	disp := FormatDisplayFromFIO(lastName, firstName, patronymic)
	if disp == "" {
		disp = strings.TrimSpace(username)
	}
	un := strings.TrimSpace(username)
	err = tx.QueryRow(ctx, `
		INSERT INTO authn.console_users (
			email, password_hash, display_name,
			first_name, last_name, patronymic, username
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		normEmail(email),
		hash,
		strings.TrimSpace(disp),
		strings.TrimSpace(firstName),
		strings.TrimSpace(lastName),
		strings.TrimSpace(patronymic),
		un,
	).Scan(&consoleUserID)
	if err != nil {
		return 0, err
	}
	cmd, err := tx.Exec(ctx, `DELETE FROM authn.admins WHERE id = $1`, adminID)
	if err != nil {
		return 0, err
	}
	if cmd.RowsAffected() == 0 {
		return 0, ErrNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return consoleUserID, nil
}
