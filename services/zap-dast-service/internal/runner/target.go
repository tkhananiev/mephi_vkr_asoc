package runner

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type targetURLValidator func(raw string) (*url.URL, error)

func validateTargetURL(raw string) (*url.URL, error) {
	return validateTargetURLWithLookup(context.Background(), raw, net.DefaultResolver.LookupIPAddr)
}

func validateTargetURLWithLookup(ctx context.Context, raw string, lookup func(context.Context, string) ([]net.IPAddr, error)) (*url.URL, error) {
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
	host := strings.TrimSpace(strings.ToLower(u.Hostname()))
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return nil, fmt.Errorf("target_url host required")
	}
	if host == "localhost" || host == "metadata.google.internal" {
		return nil, fmt.Errorf("disallowed host %q", host)
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if disallowedTargetIP(addr.Unmap()) {
			return nil, fmt.Errorf("disallowed host %q", host)
		}
		return u, nil
	}
	ips, err := lookup(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve target host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("resolve target host %q: no addresses", host)
	}
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok {
			return nil, fmt.Errorf("resolve target host %q: invalid address", host)
		}
		if disallowedTargetIP(addr.Unmap()) {
			return nil, fmt.Errorf("disallowed host %q resolves to %s", host, addr.String())
		}
	}
	return u, nil
}

func disallowedTargetIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return true
	}
	return !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified()
}

// assertTargetRedirectChainSafe follows redirects with a short GET and re-validates
// every hop. Real zap-baseline.py will follow redirects itself; without this preflight
// a public attacker host can 302 into link-local/metadata/private addresses.
func assertTargetRedirectChainSafe(ctx context.Context, targetURL string) error {
	return assertTargetRedirectChainSafeWithValidate(ctx, targetURL, validateTargetURL)
}

func assertTargetRedirectChainSafeWithValidate(ctx context.Context, targetURL string, validate targetURLValidator) error {
	if validate == nil {
		validate = validateTargetURL
	}
	u, err := validate(targetURL)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if _, err := validate(req.URL.String()); err != nil {
				return fmt.Errorf("redirect target rejected: %w", err)
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "asoc-zap-dast-preflight/1.0")
	resp, err := client.Do(req)
	if err != nil {
		// Connection failures after a safe hop are OK — ZAP will surface them.
		// Reject only when a redirect hop itself was disallowed.
		if strings.Contains(err.Error(), "redirect target rejected") ||
			strings.Contains(err.Error(), "too many redirects") {
			return err
		}
		return nil
	}
	defer resp.Body.Close()
	return nil
}
