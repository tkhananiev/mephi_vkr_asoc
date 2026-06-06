package ops

import "context"

type Backend interface {
	Logs(ctx context.Context, serviceID string, tail int) ([]byte, error)
	Restart(ctx context.Context, serviceID string) ([]byte, error)
}
