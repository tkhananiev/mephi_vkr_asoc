package runner

import (
	"fmt"
	"net/url"
	"strings"
)

func validateTargetURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("target_url required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return nil, fmt.Errorf("target_url scheme must be http or https")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("target_url host required")
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" ||
		strings.HasPrefix(host, "0.") || host == "metadata.google.internal" {
		return nil, fmt.Errorf("disallowed host %q", host)
	}
	return u, nil
}
