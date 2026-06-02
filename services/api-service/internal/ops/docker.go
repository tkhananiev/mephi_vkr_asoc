package ops

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Имена контейнеров из deploy/compose.yaml (локально).
var composeContainerByService = map[string]string{
	"api":  "mephi-vkr-api-service",
	"auth": "mephi-vkr-auth-service",
	"ref":  "mephi-vkr-reference-data-service",
	"prc":  "mephi-vkr-processing-service",
	"jir":  "mephi-vkr-jira-integration-service",
	"sem":  "mephi-vkr-semgrep-service",
	"gls":  "mephi-vkr-gitleaks-service",
	"sca":  "mephi-vkr-trivy-sca-service",
	"trivy": "mephi-vkr-trivy-sca-service",
	"zap":  "mephi-vkr-zap-dast-service",
	"dast": "mephi-vkr-zap-dast-service",
}

const maxLogBytes = 512 * 1024

// Runner выполняет docker logs / docker restart только для контейнеров из белого списка.
type Runner struct {
	CLI string
}

func NewRunner(cli string) *Runner {
	cli = strings.TrimSpace(cli)
	if cli == "" {
		cli = "docker"
	}
	return &Runner{CLI: cli}
}

func (r *Runner) ResolveService(serviceID string) (container string, ok bool) {
	if r == nil {
		return "", false
	}
	id := strings.ToLower(strings.TrimSpace(serviceID))
	c, ok := composeContainerByService[id]
	return c, ok
}

func (r *Runner) Logs(ctx context.Context, serviceID string, tail int) ([]byte, error) {
	container, ok := r.ResolveService(serviceID)
	if !ok {
		return nil, fmt.Errorf("unknown service id")
	}
	if tail < 1 {
		tail = 200
	}
	if tail > 2000 {
		tail = 2000
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.CLI, "logs", "--tail", fmt.Sprintf("%d", tail), container)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return out, fmt.Errorf("%w: %s", err, string(out))
		}
		return nil, err
	}
	if len(out) > maxLogBytes {
		trunc := []byte("(показан хвост лога, объём ограничен)\n\n")
		out = append(trunc, out[len(out)-maxLogBytes:]...)
	}
	return out, nil
}

func (r *Runner) Restart(ctx context.Context, serviceID string) ([]byte, error) {
	container, ok := r.ResolveService(serviceID)
	if !ok {
		return nil, fmt.Errorf("unknown service id")
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.CLI, "restart", container)
	return cmd.CombinedOutput()
}

var _ Backend = (*Runner)(nil)
