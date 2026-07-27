package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct {
	binary string
	config string
}

func New(binary, defaultConfig string) *Runner {
	return &Runner{binary: binary, config: defaultConfig}
}

// ValidateSemgrepConfig rejects remote URLs, absolute paths, and flag-like values.
// Semgrep fetches http(s) --config targets, which would otherwise be an SSRF primitive.
func ValidateSemgrepConfig(cfg string) error {
	cfg = strings.TrimSpace(cfg)
	if cfg == "" {
		return fmt.Errorf("empty semgrep config")
	}
	if strings.HasPrefix(cfg, "-") {
		return fmt.Errorf("semgrep_config must not start with '-'")
	}
	lower := strings.ToLower(cfg)
	if strings.Contains(lower, "://") || strings.HasPrefix(lower, "file:") {
		return fmt.Errorf("semgrep_config must not be a URL")
	}
	if filepath.IsAbs(cfg) {
		return fmt.Errorf("semgrep_config must not be an absolute path")
	}
	cleaned := filepath.ToSlash(filepath.Clean(cfg))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("semgrep_config path escapes scan tree")
	}
	return nil
}

// (некоторые версии semgrep завершаются с ошибкой при наличии находок).
func (r *Runner) Run(ctx context.Context, targetPath, configOverride string) ([]byte, error) {
	cfg := r.config
	if strings.TrimSpace(configOverride) != "" {
		cfg = strings.TrimSpace(configOverride)
	}
	if err := ValidateSemgrepConfig(cfg); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, r.binary, "scan", "--config", cfg, "--json", targetPath)
	output, cmdErr := cmd.CombinedOutput()

	jsonStart := strings.Index(string(output), "{")
	if jsonStart < 0 {
		if cmdErr != nil {
			return nil, fmt.Errorf("semgrep failed: %w; output=%s", cmdErr, string(output))
		}
		return nil, fmt.Errorf("semgrep output does not contain json payload: %s", string(output))
	}

	dec := json.NewDecoder(bytes.NewReader(output[jsonStart:]))
	dec.UseNumber()
	var payload json.RawMessage
	if err := dec.Decode(&payload); err != nil {
		if cmdErr != nil {
			return nil, fmt.Errorf("semgrep: %w; output=%s", cmdErr, string(output))
		}
		return nil, fmt.Errorf("decode semgrep json: %w", err)
	}

	if cmdErr != nil {
	
		return payload, nil
	}
	return payload, nil
}
