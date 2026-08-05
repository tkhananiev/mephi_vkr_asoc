package runner

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

// testValidate allows loopback httptest targets but still blocks metadata/link-local literals.
func testValidate(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	host := strings.ToLower(u.Hostname())
	if host == "169.254.169.254" || host == "metadata.google.internal" || strings.HasPrefix(host, "169.254.") {
		return nil, fmt.Errorf("disallowed host %q", host)
	}
	return u, nil
}

func TestAssertTargetRedirectChainSafeRejectsMetadataRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	err := assertTargetRedirectChainSafeWithValidate(context.Background(), srv.URL+"/", testValidate)
	if err == nil {
		t.Fatal("expected redirect to metadata IP to be rejected")
	}
	if !strings.Contains(err.Error(), "redirect target rejected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertTargetRedirectChainSafeAllowsPublicRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(final.Close)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/", http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	if err := assertTargetRedirectChainSafeWithValidate(context.Background(), srv.URL+"/", testValidate); err != nil {
		t.Fatalf("public redirect chain rejected: %v", err)
	}
}
