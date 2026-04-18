package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// connectPostgresWithRetry ждёт готовности PostgreSQL при старте compose (depends_on не ждёт accept на порту).
func connectPostgresWithRetry(parentCtx context.Context, dsn string) (*pgxpool.Pool, error) {
	const maxWait = 90 * time.Second
	start := time.Now()
	backoff := 500 * time.Millisecond
	var lastErr error
	logLabel := "reference-data-service"

	for time.Since(start) < maxWait {
		select {
		case <-parentCtx.Done():
			if lastErr != nil {
				return nil, fmt.Errorf("%w (last postgres error: %v)", parentCtx.Err(), lastErr)
			}
			return nil, parentCtx.Err()
		default:
		}

		pctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		pool, err := pgxpool.New(pctx, dsn)
		cancel()
		if err != nil {
			lastErr = err
			if !isPostgresTransientErr(err) {
				return nil, fmt.Errorf("connect postgres: %w", err)
			}
			log.Printf("%s: postgres not ready: %v (retry in %v)", logLabel, err, backoff)
		} else {
			pingCtx, pcancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = pool.Ping(pingCtx)
			pcancel()
			if err == nil {
				return pool, nil
			}
			lastErr = err
			pool.Close()
			if !isPostgresTransientErr(err) {
				return nil, fmt.Errorf("ping postgres: %w", err)
			}
			log.Printf("%s: postgres ping failed: %v (retry in %v)", logLabel, err, backoff)
		}

		select {
		case <-parentCtx.Done():
			return nil, fmt.Errorf("%w: %v", parentCtx.Err(), lastErr)
		case <-time.After(backoff):
		}
		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
	return nil, fmt.Errorf("postgres unavailable after %s: %w", maxWait, lastErr)
}

func isPostgresTransientErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "no such host") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "eof") ||
		strings.Contains(s, "starting up") ||
		strings.Contains(s, "recovery") ||
		strings.Contains(s, "the database system is starting up")
}
