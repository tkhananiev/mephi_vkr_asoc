package service

import "testing"

func TestCanonicalIngestScannerNameIgnoresClientSpoof(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		executedID  string
		catalogName string
		want        string
	}{
		{name: "gitleaks route ignores semgrep spoof", executedID: "gitleaks", want: "gitleaks"},
		{name: "semgrep route", executedID: "semgrep", want: "semgrep"},
		{name: "sca alias", executedID: "sca", want: "trivy-sca"},
		{name: "trivy alias", executedID: "trivy", want: "trivy-sca"},
		{name: "dast alias", executedID: "dast", want: "zap-dast"},
		{name: "zap alias", executedID: "zap", want: "zap-dast"},
		{name: "dynamic prefers catalog", executedID: "custom-tool", catalogName: "my-scanner", want: "my-scanner"},
		{name: "dynamic falls back to id", executedID: "custom-tool", catalogName: "", want: "custom-tool"},
		{name: "case insensitive built-in", executedID: "GiTLeAkS", want: "gitleaks"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := canonicalIngestScannerName(tc.executedID, tc.catalogName)
			if got != tc.want {
				t.Fatalf("canonicalIngestScannerName(%q, %q) = %q, want %q", tc.executedID, tc.catalogName, got, tc.want)
			}
		})
	}
}
