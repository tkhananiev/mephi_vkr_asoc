package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// runHTTPProbeStub — лёгкая проверка доступности цели без OWASP ZAP (APP_ZAP_USE_STUB=true).
func runHTTPProbeStub(ctx context.Context, targetURL string) ([]byte, error) {
	u, err := validateTargetURL(targetURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 45 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if _, err := validateTargetURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "asoc-zap-dast-stub/1.0")

	resp, err := client.Do(req)
	findings := make([]normalizedFinding, 0, 2)
	host := u.Hostname()

	if err != nil {
		findings = append(findings, normalizedFinding{
			AssetID:    host,
			Identifier: "dast-connect-failed",
			Severity:   "high",
			Component:  u.String(),
			Metadata: map[string]any{
				"title":   "HTTP target unreachable",
				"message": err.Error(),
				"engine":  "zap-dast-stub",
			},
			RawPayload: map[string]any{"target_url": u.String()},
		})
		return encodeFindings(findings)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		sev := "medium"
		if resp.StatusCode >= 500 {
			sev = "high"
		}
		findings = append(findings, normalizedFinding{
			AssetID:    host,
			Identifier: fmt.Sprintf("dast-http-%d", resp.StatusCode),
			Severity:   sev,
			Component:  u.String(),
			Metadata: map[string]any{
				"title":       fmt.Sprintf("HTTP %d from target", resp.StatusCode),
				"status_code": resp.StatusCode,
				"engine":      "zap-dast-stub",
			},
			RawPayload: map[string]any{"target_url": u.String(), "status_code": resp.StatusCode},
		})
	}

	return encodeFindings(findings)
}
