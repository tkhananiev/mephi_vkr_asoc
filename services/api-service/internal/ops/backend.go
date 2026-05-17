package ops

import "context"

// Backend — логи и рестарт «логического» сервиса (docker-compose или Kubernetes).
type Backend interface {
	Logs(ctx context.Context, serviceID string, tail int) ([]byte, error)
	Restart(ctx context.Context, serviceID string) ([]byte, error)
}
