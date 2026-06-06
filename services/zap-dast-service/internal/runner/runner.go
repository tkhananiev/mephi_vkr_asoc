package runner

import (
	"context"
)

type Runner struct {
	zapHome    string
	timeoutMin int
	useStub    bool
}

func New(zapHome string, timeoutMin int, useStub bool) *Runner {
	if timeoutMin < 1 {
		timeoutMin = 8
	}
	return &Runner{
		zapHome:    zapHome,
		timeoutMin: timeoutMin,
		useStub:    useStub,
	}
}

func (r *Runner) Run(ctx context.Context, targetURL string) ([]byte, error) {
	if r.useStub {
		return runHTTPProbeStub(ctx, targetURL)
	}
	return runZAPBaseline(ctx, r.zapHome, targetURL, r.timeoutMin)
}
