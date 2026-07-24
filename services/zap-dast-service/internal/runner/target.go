package runner

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

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
