package httpapi

import (
	"context"
	"net/http"
)

type consoleUserKey struct{}
type adminUserKey struct{}

func WithConsoleUser(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, consoleUserKey{}, userID)
}

func WithAdminUser(ctx context.Context, adminID int64) context.Context {
	return context.WithValue(ctx, adminUserKey{}, adminID)
}

func ConsoleUserFromRequest(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(consoleUserKey{}).(int64)
	return id, ok
}

func AdminUserFromRequest(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(adminUserKey{}).(int64)
	return id, ok
}
