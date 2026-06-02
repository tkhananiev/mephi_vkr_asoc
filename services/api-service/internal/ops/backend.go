package ops

import "context"

// Backend — логи и рестарт логического сервиса (локально или в оркестрируемой среде).
type Backend interface {
	Logs(ctx context.Context, serviceID string, tail int) ([]byte, error)
	Restart(ctx context.Context, serviceID string) ([]byte, error)
}
