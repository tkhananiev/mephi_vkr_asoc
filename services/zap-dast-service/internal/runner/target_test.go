package runner

import (
	"context"
	"net"
	"testing"
)

func TestValidateTargetURLRejectsMetadataAndPrivate(t *testing.T) {
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("169.254.169.254")}}, nil
	}
	cases := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5/",
		"http://127.0.0.1/",
		"http://metadata.google.internal/",
	}
	for _, raw := range cases {
		if _, err := validateTargetURLWithLookup(context.Background(), raw, lookup); err == nil {
			t.Fatalf("expected reject for %q", raw)
		}
	}
	if _, err := validateTargetURLWithLookup(context.Background(), "https://example.com/app", func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	}); err != nil {
		t.Fatalf("public target rejected: %v", err)
	}
}
