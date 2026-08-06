package integrationstore

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// ValidateScannerInvokeURL ensures scanner_invoke_url is http(s), has no userinfo,
// and does not target loopback, link-local, or cloud-metadata addresses.
// RFC1918 / ULA cluster addresses remain allowed so in-cluster runners
// (e.g. http://generic-scan-runner:8087/api/v1/run) keep working.
func ValidateScannerInvokeURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("scanner_invoke_url is empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("scanner_invoke_url must be absolute URL with http or https scheme")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scanner_invoke_url must be absolute URL with http or https scheme")
	}
	if parsed.Host == "" {
		return fmt.Errorf("scanner_invoke_url must be absolute URL with http or https scheme")
	}
	if parsed.User != nil {
		return fmt.Errorf("scanner_invoke_url must not contain userinfo")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return fmt.Errorf("scanner_invoke_url host required")
	}
	if isMetadataHostname(host) {
		return fmt.Errorf("scanner_invoke_url host %q is not allowed", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("scanner_invoke_url host %q could not be resolved: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("scanner_invoke_url host %q resolved to no addresses", host)
	}
	for _, a := range addrs {
		if err := rejectDisallowedInvokeIP(a.IP); err != nil {
			return fmt.Errorf("scanner_invoke_url host %q: %w", host, err)
		}
	}
	return nil
}

func isMetadataHostname(host string) bool {
	switch host {
	case "metadata", "metadata.google.internal", "metadata.goog":
		return true
	default:
		return false
	}
}

func rejectDisallowedInvokeIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("invalid IP")
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("address %s is not allowed", ip.String())
	}
	if ip.IsLinkLocalUnicast() || isCloudMetadataIP(ip) {
		return fmt.Errorf("address %s is not allowed", ip.String())
	}
	return nil
}

func isCloudMetadataIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		// 169.254.0.0/16 — link-local / cloud IMDS
		return ip4[0] == 169 && ip4[1] == 254
	}
	// AWS IMDSv2 IPv6 (ULA, not link-local)
	return ip.Equal(net.ParseIP("fd00:ec2::254"))
}
