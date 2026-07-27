package runner

import "testing"

func TestValidateSemgrepConfig(t *testing.T) {
	t.Parallel()
	ok := []string{"p/java", "r/python", "auto", "rules/custom.yaml", "./rules/local.yml"}
	for _, cfg := range ok {
		if err := ValidateSemgrepConfig(cfg); err != nil {
			t.Fatalf("expected ok for %q: %v", cfg, err)
		}
	}
	bad := []string{
		"",
		"--jobs=64",
		"https://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/rules.yaml",
		"file:///etc/passwd",
		"/etc/passwd",
		"/var/run/secrets/kubernetes.io/serviceaccount/token",
		"../../../etc/passwd",
	}
	for _, cfg := range bad {
		if err := ValidateSemgrepConfig(cfg); err == nil {
			t.Fatalf("expected error for %q", cfg)
		}
	}
}
