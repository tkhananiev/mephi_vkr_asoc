package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// WithAPIKeyAuth оборачивает маршрутизатор: при непустом apiKey требует ключ для /api/*.
// Без ключа остаются /health, /openapi.yaml, /swagger (probes K8s и UI документации).
func WithAPIKeyAuth(apiKey string, next http.Handler) http.Handler {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return next
	}
	want := []byte(key)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/health" || p == "/openapi.yaml" {
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
		if !matchSecret(r, want) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="aspm"`)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func matchSecret(r *http.Request, want []byte) bool {
	const pfx = "bearer "
	a := r.Header.Get("Authorization")
	if len(a) > 7 && strings.ToLower(a[:7]) == pfx {
		g := []byte(strings.TrimSpace(a[7:]))
		return len(g) == len(want) && subtle.ConstantTimeCompare(g, want) == 1
	}
	x := []byte(strings.TrimSpace(r.Header.Get("X-API-Key")))
	return len(x) == len(want) && subtle.ConstantTimeCompare(x, want) == 1
}
