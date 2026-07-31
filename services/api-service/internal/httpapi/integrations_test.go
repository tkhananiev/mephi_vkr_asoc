package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mephi_vkr_asoc/services/api-service/internal/integrationstore"
)

func TestHandleIntegrationsListOmitsSecrets(t *testing.T) {
	store, err := integrationstore.New("")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceAdditional([]integrationstore.Item{{
		ID:                    "secret-scanner",
		Kind:                  "SAST",
		Title:                 "Secret Scanner",
		Summary:               "Has secrets",
		Phase:                 "ready",
		Enabled:               true,
		InputKind:             "filesystem",
		ScannerName:           "secret-scanner",
		ScannerInvokeURL:      "https://runner.example/api/v1/run",
		RunnerCommand:         "tool --token SUPERSECRET {target_path}",
		InvokePayloadTemplate: `{"token":"SUPERSECRET"}`,
		NetworkIP:             "192.168.1.10",
	}}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{integrationStore: store}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	rr := httptest.NewRecorder()
	h.handleIntegrationsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}

	var resp integrationsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	body := rr.Body.String()
	for _, leak := range []string{
		"scanner_invoke_url",
		"runner_command",
		"invoke_payload_template",
		"network_ip",
		"SUPERSECRET",
		"192.168.1.10",
		"runner.example",
	} {
		if strings.Contains(body, leak) {
			t.Fatalf("response leaked %q: %s", leak, body)
		}
	}

	found := false
	for _, it := range resp.Integrations {
		if it.ID == "secret-scanner" {
			found = true
			if it.Title != "Secret Scanner" || !it.Enabled {
				t.Fatalf("public fields wrong: %+v", it)
			}
		}
	}
	if !found {
		t.Fatal("expected secret-scanner in public list")
	}
}
