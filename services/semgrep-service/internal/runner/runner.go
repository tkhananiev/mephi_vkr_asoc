package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct {
	binary string
	config string
}

func New(binary, defaultConfig string) *Runner {
	return &Runner{binary: binary, config: defaultConfig}
}

// (некоторые версии semgrep завершаются с ошибкой при наличии находок).
func (r *Runner) Run(ctx context.Context, targetPath, configOverride string) ([]byte, error) {
	cfg := r.config
	if strings.TrimSpace(configOverride) != "" {
		cfg = strings.TrimSpace(configOverride)
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
