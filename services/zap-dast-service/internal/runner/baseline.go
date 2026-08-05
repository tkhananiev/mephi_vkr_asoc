package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func runZAPBaseline(ctx context.Context, zapHome, targetURL string, timeoutMin int) ([]byte, error) {
	if _, err := validateTargetURL(targetURL); err != nil {
		return nil, err
	}
	if err := assertTargetRedirectChainSafe(ctx, targetURL); err != nil {
		return nil, err
	}
	home := strings.TrimSpace(zapHome)
	if home == "" {
		home = "/zap"
	}
	script := filepath.Join(home, "zap-baseline.py")
	if _, err := os.Stat(script); err != nil {
		return nil, fmt.Errorf("ZAP baseline script not found at %s (set APP_ZAP_HOME or use APP_ZAP_USE_STUB=true)", script)
	}

	reportDir, err := os.MkdirTemp("", "asoc-zap-report-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(reportDir)

	reportPath := filepath.Join(reportDir, "report.json")
	args := []string{
		script,
		"-t", targetURL,
		"-J", reportPath,
		"-I",
	}
	if timeoutMin > 0 {
		args = append(args, "-T", fmt.Sprintf("%d", timeoutMin))
	}

	runCtx := ctx
	if timeoutMin > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMin+2)*time.Minute)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, "python3", args...)
	cmd.Dir = home
	var stderr strings.Builder
	cmd.Stderr = &stderr
	_ = cmd.Run()

	raw, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		hint := strings.TrimSpace(stderr.String())
		if hint != "" {
			return nil, fmt.Errorf("ZAP baseline produced no JSON report: %v; stderr: %s", readErr, hint)
		}
		return nil, fmt.Errorf("ZAP baseline produced no JSON report: %w", readErr)
	}

	return raw, nil
}
