package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"mephi_vkr_asoc/services/auth-service/internal/repo"
	"mephi_vkr_asoc/services/auth-service/internal/token"
)

var reConsoleUsername = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,64}$`)

type SMTPConfig struct {
	Host     string
	Port     string
	Login    string
	Password string
	From     string
}

type Handler struct {
	r      *repo.Repo
	jwtSec []byte
	jwtTTL time.Duration
	smtp   SMTPConfig
}

func New(r *repo.Repo, jwtSecret []byte, ttl time.Duration, smtpCfg SMTPConfig) *Handler {
	return &Handler{r: r, jwtSec: jwtSecret, jwtTTL: ttl, smtp: smtpCfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /auth/v1/login", h.userLogin)
	mux.HandleFunc("POST /auth/v1/register", h.publicRegister)
	mux.HandleFunc("POST /auth/v1/register/verify", h.verifyRegistrationCode)
	mux.HandleFunc("POST /auth/v1/admin/login", h.adminLogin)
	mux.HandleFunc("GET /auth/v1/admin/users", h.listAccounts)
	mux.HandleFunc("POST /auth/v1/admin/users", h.createAccount)
	mux.HandleFunc("PATCH /auth/v1/admin/console-users/{id}", h.patchConsoleUser)
	mux.HandleFunc("POST /auth/v1/admin/console-users/{id}/password", h.resetConsoleUserPassword)
	mux.HandleFunc("DELETE /auth/v1/admin/console-users/{id}", h.deleteConsoleUser)
	mux.HandleFunc("POST /auth/v1/admin/console-users/{id}/promote-to-admin", h.promoteConsoleToAdmin)
	mux.HandleFunc("PATCH /auth/v1/admin/admins/{id}", h.patchAdmin)
	mux.HandleFunc("POST /auth/v1/admin/admins/{id}/password", h.resetAdminPassword)
	mux.HandleFunc("DELETE /auth/v1/admin/admins/{id}", h.deleteAdmin)
	mux.HandleFunc("POST /auth/v1/admin/admins/{id}/demote-to-console-user", h.demoteAdminToConsoleUser)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) requireAdmin(r *http.Request) (*token.Claims, error) {
	a := r.Header.Get("Authorization")
	if len(a) < 8 || !strings.EqualFold(a[:7], "bearer ") {
		return nil, errors.New("no bearer")
	}
	tok := strings.TrimSpace(a[7:])
	c, err := token.Parse(h.jwtSec, tok)
	if err != nil || c.Role != "admin" {
		return nil, errors.New("forbidden")
	}
	return c, nil
}

func normalizeConsoleIdentifier(identifierField, legacyEmailField string) string {
	who := strings.TrimSpace(identifierField)
	if who == "" {
		who = strings.TrimSpace(legacyEmailField)
	}
	return who
}

// validateConsoleUserProfile — общие правила для пользователя консоли (ФИО, логин, e-mail): саморегистрация, создание админом, понижение админа.
func validateConsoleUserProfile(lastName, firstName, patronymic, username, email string) string {
	if msg := validateChunk("last_name", lastName, 120, true); msg != "" {
		return msg
	}
	if msg := validateChunk("first_name", firstName, 120, true); msg != "" {
		return msg
	}
	if msg := validateChunk("patronymic", patronymic, 120, false); msg != "" {
		return msg
	}
	em := strings.TrimSpace(email)
	if em == "" || !strings.Contains(em, "@") {
		return "valid email is required"
	}
	user := strings.TrimSpace(username)
	if !reConsoleUsername.MatchString(user) {
		return "username must match [a-zA-Z0-9._-]{3,64}"
	}
	return ""
}

func validateConsolePasswordMin8(pw string) string {
	if utf8.RuneCountInString(pw) < 8 {
		return "password must be at least 8 characters"
	}
	return ""
}

func validateConsoleUserSignup(lastName, firstName, patronymic, username, email, password string) string {
	if msg := validateConsoleUserProfile(lastName, firstName, patronymic, username, email); msg != "" {
		return msg
	}
	return validateConsolePasswordMin8(password)
}

func parseRegistrationVerifyInput(emailRaw, codeRaw string) (email string, code string, errMsg string) {
	em := strings.TrimSpace(emailRaw)
	code = strings.TrimSpace(codeRaw)
	if em == "" || !strings.Contains(em, "@") {
		return "", "", "valid email is required"
	}
	if len(code) != 6 {
		return "", "", "verification code must be 6 digits"
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return "", "", "verification code must be 6 digits"
		}
	}
	return em, code, ""
}

func (h *Handler) userLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Identifier string `json:"identifier"`
		Email      string `json:"email"` // устарело: клиент должен слать только identifier; оставлено для совместимости
		Password   string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	who := normalizeConsoleIdentifier(body.Identifier, body.Email)
	if who == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "identifier is required"})
		return
	}

	a, adminErr := h.r.GetAdminByLogin(r.Context(), who)
	switch {
	case adminErr == nil:
		if a.Disabled || bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(body.Password)) != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid login or password"})
			return
		}
		tok, err := token.Issue(h.jwtSec, a.ID, a.Login, "", "admin", h.jwtTTL)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": tok,
			"expires_in":   int(h.jwtTTL.Seconds()),
			"token_type":   "Bearer",
			"role":         "admin",
		})
		return
	case errors.Is(adminErr, repo.ErrNotFound):
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": adminErr.Error()})
		return
	}

	u, err := h.r.GetConsoleUserByLogin(r.Context(), who)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	if u.Disabled || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	tok, err := token.Issue(h.jwtSec, u.ID, u.Email, u.DisplayName, "user", h.jwtTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"expires_in":   int(h.jwtTTL.Seconds()),
		"token_type":   "Bearer",
		"role":         "user",
	})
}

func (h *Handler) publicRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email      string `json:"email"`
		Username   string `json:"username"`
		Password   string `json:"password"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		Patronymic string `json:"patronymic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if msg := validateConsoleUserSignup(body.LastName, body.FirstName, body.Patronymic, body.Username, body.Email, body.Password); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	em := strings.TrimSpace(body.Email)
	user := strings.TrimSpace(body.Username)
	if strings.TrimSpace(h.smtp.Password) == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "smtp is not configured"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	code, err := makeVerificationCode()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot generate verification code"})
		return
	}
	if err := h.r.UpsertPendingRegistration(r.Context(), repo.PendingRegistration{
		Email: em, Username: user, FirstName: strings.TrimSpace(body.FirstName), LastName: strings.TrimSpace(body.LastName),
		Patronymic: strings.TrimSpace(body.Patronymic), PasswordHash: string(hash), VerificationCode: code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := h.sendVerificationCode(em, code); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to send verification email"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":     "verification_sent",
		"expires_in": 900,
	})
}

func (h *Handler) verifyRegistrationCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	em, code, parseErr := parseRegistrationVerifyInput(body.Email, body.Code)
	if parseErr != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": parseErr})
		return
	}
	p, err := h.r.GetPendingRegistration(r.Context(), em)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verification not found"})
		return
	}
	if time.Now().After(p.ExpiresAt) {
		_ = h.r.DeletePendingRegistration(r.Context(), em)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verification code expired"})
		return
	}
	if strings.TrimSpace(p.VerificationCode) != code {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid verification code"})
		return
	}
	id, err := h.r.InsertConsoleUser(r.Context(), repo.ConsoleUserInsert{
		Email: p.Email, Username: p.Username, FirstName: p.FirstName, LastName: p.LastName,
		Patronymic: p.Patronymic, PasswordHash: p.PasswordHash,
	})
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email or username already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_ = h.r.DeletePendingRegistration(r.Context(), em)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "kind": "user"})
}

func (h *Handler) sendVerificationCode(email, code string) error {
	addr := strings.TrimSpace(h.smtp.Host) + ":" + strings.TrimSpace(h.smtp.Port)
	auth := smtp.PlainAuth("", strings.TrimSpace(h.smtp.Login), strings.TrimSpace(h.smtp.Password), strings.TrimSpace(h.smtp.Host))
	from := strings.TrimSpace(h.smtp.From)
	if from == "" {
		from = "no-reply@atomic-asoc.ru"
	}
	subject := "Atomic ASOC: код подтверждения регистрации"
	body := fmt.Sprintf("Код подтверждения: %s\n\nКод действует 15 минут.\n", code)
	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + email + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" + body + "\r\n",
	)
	return smtp.SendMail(addr, auth, from, []string{email}, msg)
}

func makeVerificationCode() (string, error) {
	const digits = "0123456789"
	buf := make([]byte, 6)
	raw := make([]byte, 6)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i := range raw {
		buf[i] = digits[int(raw[i])%10]
	}
	return string(buf), nil
}

func validateChunk(label string, val string, maxRunes int, required bool) string {
	v := strings.TrimSpace(val)
	if required && v == "" {
		switch label {
		case "first_name":
			return "Укажите имя"
		case "last_name":
			return "Укажите фамилию"
		default:
			return "Не заполнено обязательное поле"
		}
	}
	if utf8.RuneCountInString(v) > maxRunes {
		return "Слишком длинное значение поля"
	}
	return ""
}

func (h *Handler) adminLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	a, err := h.r.GetAdminByLogin(r.Context(), body.Login)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid login or password"})
		return
	}
	if a.Disabled {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid login or password"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(body.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid login or password"})
		return
	}
	tok, err := token.Issue(h.jwtSec, a.ID, a.Login, "", "admin", h.jwtTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"expires_in":   int(h.jwtTTL.Seconds()),
		"token_type":   "Bearer",
		"role":         "admin",
	})
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	users, err := h.r.ListConsoleUsers(r.Context(), 300)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	admins, err := h.r.ListAdmins(r.Context(), 300)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	type jsonAccount struct {
		Kind        string    `json:"kind"`
		StableID    string    `json:"stable_id"`
		ID          int64     `json:"id"`
		Email       string    `json:"email,omitempty"`
		Username    string    `json:"username,omitempty"`
		FirstName   string    `json:"first_name,omitempty"`
		LastName    string    `json:"last_name,omitempty"`
		Patronymic  string    `json:"patronymic,omitempty"`
		DisplayName string    `json:"display_name,omitempty"`
		Login       string    `json:"login,omitempty"`
		Disabled    bool      `json:"disabled"`
		CreatedAt   string    `json:"created_at"`
		TS          time.Time `json:"-"`
	}
	out := make([]jsonAccount, 0, len(users)+len(admins))
	for _, u := range users {
		out = append(out, jsonAccount{
			Kind: "user", StableID: fmt.Sprintf("CU-%d", u.ID), ID: u.ID, Email: u.Email, Username: u.Username,
			FirstName: u.FirstName, LastName: u.LastName, Patronymic: u.Patronymic,
			DisplayName: u.DisplayName, Disabled: u.Disabled,
			CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339), TS: u.CreatedAt,
		})
	}
	for _, a := range admins {
		out = append(out, jsonAccount{
			Kind: "admin", StableID: fmt.Sprintf("ADM-%d", a.ID), ID: a.ID, Login: a.Login, Disabled: a.Disabled,
			CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339), TS: a.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS.After(out[j].TS) })
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var body struct {
		Role       string `json:"role"`
		Email      string `json:"email"`
		Username   string `json:"username"`
		Login      string `json:"login"`
		Password   string `json:"password"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		Patronymic string `json:"patronymic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	role := strings.ToLower(strings.TrimSpace(body.Role))
	if role == "" {
		role = "user"
	}
	if msg := validateConsolePasswordMin8(body.Password); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	switch role {
	case "user":
		em := strings.TrimSpace(body.Email)
		user := strings.TrimSpace(body.Username)
		if msg := validateConsoleUserProfile(body.LastName, body.FirstName, body.Patronymic, user, em); msg != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}
		id, err := h.r.InsertConsoleUser(r.Context(), repo.ConsoleUserInsert{
			Email: em, Username: user, FirstName: strings.TrimSpace(body.FirstName),
			LastName: strings.TrimSpace(body.LastName), Patronymic: strings.TrimSpace(body.Patronymic), PasswordHash: string(hash),
		})
		if err != nil {
			var pe *pgconn.PgError
			if errors.As(err, &pe) && pe.Code == "23505" {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "email or username already exists"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "kind": "user"})
	case "admin":
		login := strings.TrimSpace(body.Login)
		if login == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login is required for admin role"})
			return
		}
		id, err := h.r.InsertAdmin(r.Context(), login, string(hash))
		if err != nil {
			var pe *pgconn.PgError
			if errors.As(err, &pe) && pe.Code == "23505" {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "login already exists"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "kind": "admin"})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role must be user or admin"})
	}
}

func (h *Handler) patchConsoleUser(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Email       *string `json:"email"`
		Disabled    *bool   `json:"disabled"`
		DisplayName *string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if body.DisplayName != nil && utf8.RuneCountInString(strings.TrimSpace(*body.DisplayName)) > 200 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "display_name too long"})
		return
	}
	if err := h.r.UpdateConsoleUser(r.Context(), uid, body.Email, body.Disabled, body.DisplayName); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) patchAdmin(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Login    *string `json:"login"`
		Disabled *bool   `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := h.r.UpdateAdmin(r.Context(), uid, body.Login, body.Disabled); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "login already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) resetConsoleUserPassword(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if msg := validateConsolePasswordMin8(body.Password); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := h.r.SetConsoleUserPassword(r.Context(), uid, string(hash)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) resetAdminPassword(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if msg := validateConsolePasswordMin8(body.Password); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := h.r.SetAdminPassword(r.Context(), uid, string(hash)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) deleteConsoleUser(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.r.DeleteConsoleUser(r.Context(), uid); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) deleteAdmin(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ctx := r.Context()
	nAdm, err := h.r.AdminCount(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if nAdm <= 1 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot delete the last administrator account"})
		return
	}
	if err := h.r.DeleteAdmin(ctx, uid); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "admin not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) promoteConsoleToAdmin(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	cid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || cid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Login string `json:"login"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	u, err := h.r.GetConsoleUser(r.Context(), cid)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	login := strings.TrimSpace(body.Login)
	if login == "" {
		login = strings.TrimSpace(u.Username)
	}
	if login == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login is required (пустой username — передайте login в теле запроса)"})
		return
	}
	id, err := h.r.PromoteConsoleUserToAdmin(r.Context(), cid, login)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "login already exists"})
			return
		}
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "kind": "admin"})
}

func (h *Handler) demoteAdminToConsoleUser(w http.ResponseWriter, r *http.Request) {
	if _, err := h.requireAdmin(r); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc-admin"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	uid, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ctx := r.Context()
	nAdm, err := h.r.AdminCount(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if nAdm <= 1 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot demote the last administrator"})
		return
	}
	var body struct {
		Email      string `json:"email"`
		Username   string `json:"username"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		Patronymic string `json:"patronymic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	em := strings.TrimSpace(body.Email)
	user := strings.TrimSpace(body.Username)
	if msg := validateConsoleUserProfile(body.LastName, body.FirstName, body.Patronymic, user, em); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	id, err := h.r.DemoteAdminToConsoleUser(ctx, uid,
		em, user,
		strings.TrimSpace(body.FirstName), strings.TrimSpace(body.LastName), strings.TrimSpace(body.Patronymic),
	)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email or username already exists"})
			return
		}
		if errors.Is(err, repo.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "admin not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "kind": "user"})
}
