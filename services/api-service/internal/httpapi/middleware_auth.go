package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"mephi_vkr_asoc/services/api-service/internal/auth"
)

func WithAPIKeyOrUserJWT(apiKey string, jwtSecret []byte, next http.Handler) http.Handler {
	key := strings.TrimSpace(apiKey)
	want := []byte(key)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/health" || p == "/metrics" || p == "/openapi.yaml" {
			next.ServeHTTP(w, r)
			return
		}
		if p == "/swagger" || strings.HasPrefix(p, "/swagger/") {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(p, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		nr, ok := authenticateAttachConsoleUser(r, want, jwtSecret)
		if ok {
			next.ServeHTTP(w, nr)
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer realm="asoc"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	})
}

func authenticateAttachConsoleUser(r *http.Request, wantAPIKey, jwtSecret []byte) (*http.Request, bool) {
	if x := strings.TrimSpace(r.Header.Get("X-API-Key")); x != "" && len(wantAPIKey) > 0 {
		g := []byte(x)
		if len(g) == len(wantAPIKey) && subtle.ConstantTimeCompare(g, wantAPIKey) == 1 {
			return r, true
		}
	}
	a := r.Header.Get("Authorization")
	if len(a) < len("Bearer ")+1 || !strings.EqualFold(a[:len("Bearer ")], "Bearer ") {
		return nil, false
	}
	token := strings.TrimSpace(a[len("Bearer "):])
	if token == "" {
		return nil, false
	}
	if strings.Count(token, ".") == 2 && len(jwtSecret) >= 32 {
		if c, err := auth.ParseJWT(jwtSecret, token); err == nil && c != nil && c.UserID > 0 {
			switch strings.ToLower(strings.TrimSpace(c.Role)) {
			case "user":
				return r.WithContext(WithConsoleUser(r.Context(), c.UserID)), true
			case "admin":
				return r.WithContext(WithAdminUser(r.Context(), c.UserID)), true
			default:
				return nil, false
			}
		}
	}
	if len(wantAPIKey) > 0 {
		g := []byte(token)
		if len(g) == len(wantAPIKey) && subtle.ConstantTimeCompare(g, wantAPIKey) == 1 {
			return r, true
		}
	}
	return nil, false
}
