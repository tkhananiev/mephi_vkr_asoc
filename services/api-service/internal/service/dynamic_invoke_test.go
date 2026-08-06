package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"mephi_vkr_asoc/services/api-service/internal/models"
)

func TestPostDynamicScanPayloadRejectsMetadataURL(t *testing.T) {
	o := New("http://processing", "http://jira", "http://sem", "http://gl", "http://sca", "http://dast", "http://adapt", nil, nil)
	_, err := o.postDynamicScanPayload(context.Background(), "http://169.254.169.254/latest/meta-data/", models.ScanRequest{}, "meta", "")
	if err == nil {
		t.Fatal("expected metadata invoke URL to be rejected")
	}
}

func TestDynamicHTTPClientDisablesRedirects(t *testing.T) {
	o := New("http://processing", "http://jira", "http://sem", "http://gl", "http://sca", "http://dast", "http://adapt", nil, nil)
	if o.dynamicHTTP == nil || o.dynamicHTTP.CheckRedirect == nil {
		t.Fatal("dynamicHTTP CheckRedirect not configured")
	}
	err := o.dynamicHTTP.CheckRedirect(&http.Request{}, nil)
	if err == nil || !strings.Contains(err.Error(), "redirects are not allowed") {
		t.Fatalf("expected redirect disable error, got %v", err)
	}
}
