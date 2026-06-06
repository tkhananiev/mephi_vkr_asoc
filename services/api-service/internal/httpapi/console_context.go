package httpapi

import (
	"context"
	"net/http"
)

type consoleUserKey struct{}

func WithConsoleUser(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, consoleUserKey{}, userID)
}

func ConsoleUserFromRequest(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(consoleUserKey{}).(int64)
	return id, ok
}
