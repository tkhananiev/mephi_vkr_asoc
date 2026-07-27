package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"mephi_vkr_asoc/services/auth-service/internal/config"
	"mephi_vkr_asoc/services/auth-service/internal/httpapi"
	"mephi_vkr_asoc/services/auth-service/internal/repo"
)

func main() {
	cfg := config.Load()

	dsn := strings.TrimSpace(cfg.PostgresDSN)
	if dsn == "" {
		log.Fatal("auth-service: AUTH_POSTGRES_DSN is required")
	}
	jwtBytes := []byte(strings.TrimSpace(cfg.JWTSecret))
	if len(jwtBytes) < 32 {
		log.Fatal("auth-service: AUTH_JWT_SECRET must be at least 32 bytes")
	}

	ttl, err := time.ParseDuration(strings.TrimSpace(cfg.JWTTTL))
	if err != nil || ttl <= 0 {
		ttl = 168 * time.Hour
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("auth-service: postgres: %v", err)
	}
	defer pool.Close()

	r := repo.New(pool)

	n, err := r.CountEnabledAdmins(ctx)
	if err != nil {
		log.Fatalf("auth-service: enabled admin count: %v", err)
	}
	if n == 0 {
		lg := strings.TrimSpace(cfg.BootstrapLogin)
		pw := strings.TrimSpace(cfg.BootstrapPassword)
		if lg != "" && utf8.RuneCountInString(pw) >= 8 {
			hash, herr := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
			if herr != nil {
				log.Fatalf("auth-service: bootstrap bcrypt: %v", herr)
			}
			if _, ierr := r.InsertAdmin(ctx, lg, string(hash)); ierr != nil {
				log.Fatalf("auth-service: bootstrap admin: %v", ierr)
			}
			log.Printf("auth-service: created bootstrap admin login=%q (change password after first deploy)", lg)
		} else {
			log.Printf("auth-service: no enabled admins in DB; set AUTH_BOOTSTRAP_ADMIN_LOGIN and AUTH_BOOTSTRAP_ADMIN_PASSWORD (≥8 chars) to create the first admin")
		}
	}

	h := httpapi.New(r, jwtBytes, ttl, httpapi.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Login:    cfg.SMTPLogin,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
	mux := http.NewServeMux()
	h.Register(mux)

	addr := ":" + strings.TrimSpace(cfg.HTTPPort)
	if strings.TrimSpace(cfg.HTTPPort) == "" {
		addr = ":8091"
	}
	log.Printf("auth-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
