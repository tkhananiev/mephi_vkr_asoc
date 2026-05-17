package httpapi

import (
	"context"
	"net/http"
)

type consoleUserKey struct{}

// WithConsoleUser — привязать id пользователя консоли к запросу (только JWT role=user из auth-service).
func WithConsoleUser(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, consoleUserKey{}, userID)
}

// ConsoleUserFromRequest возвращает authn.console_users.id, если в контексте был выставлен пользователь консоли.
func ConsoleUserFromRequest(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(consoleUserKey{}).(int64)
	return id, ok
}
