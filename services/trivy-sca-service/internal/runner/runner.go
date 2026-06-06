package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct {
	binary string
}

func New(binary string) *Runner {
	return &Runner{binary: strings.TrimSpace(binary)}
}

func (r *Runner) Run(ctx context.Context, targetPath string) ([]byte, error) {
	tp := filepath.Clean(strings.TrimSpace(targetPath))
	if tp == "" || tp == "." {
		return nil, fmt.Errorf("empty target path")
	}
	cmd := exec.CommandContext(ctx, r.binary,
		"fs",
		"--format", "json",
		"--scanners", "vuln",
		"--exit-code", "0",
		"--quiet",
		tp,
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(buf.String())
		if len(out) > 0 {
			return nil, fmt.Errorf("trivy: %w; output=%s", err, out)
		}
		return nil, fmt.Errorf("trivy: %w", err)
	}
	trimmed := bytes.TrimSpace(buf.Bytes())
	if len(trimmed) == 0 {
		return []byte(`{"Results":[]}`), nil
	}
	return trimmed, nil
}
