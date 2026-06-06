package runner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct {
	binary string
}

func New(binary string) *Runner {
	return &Runner{binary: binary}
}

func randomSuffix() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// чтобы отчёт можно было разобрать при наличии находок.
func (r *Runner) Run(ctx context.Context, targetPath string) ([]byte, error) {
	tp := filepath.Clean(strings.TrimSpace(targetPath))
	if tp == "" || tp == "." {
		return nil, fmt.Errorf("empty target path")
	}
	reportPath := filepath.Join(os.TempDir(), "gitleaks-report-"+randomSuffix()+".json")
	defer func() { _ = os.Remove(reportPath) }()

	cmd := exec.CommandContext(ctx, r.binary,
		"detect",
		"--source", tp,
		"--report-format", "json",
		"--report-path", reportPath,
		"--exit-code", "0",
	)
	output, cmdErr := cmd.CombinedOutput()

	data, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		if cmdErr != nil {
			return nil, fmt.Errorf("gitleaks: %w; output=%s", cmdErr, string(output))
		}
		return nil, fmt.Errorf("gitleaks report: %w; output=%s", readErr, string(output))
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return []byte("[]"), nil
	}
	return trimmed, nil
}
