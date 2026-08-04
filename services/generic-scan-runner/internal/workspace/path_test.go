package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssertPathUnderAnyRootAllowsConfiguredRoots(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "proj")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}

	got, err := AssertPathUnderAnyRoot(nested, []string{"/var/run/secrets", root})
	if err != nil {
		t.Fatalf("expected allow under configured root: %v", err)
	}
	if got != nested {
		// EvalSymlinks may change the path; ensure still under root.
		if !isUnder(root, got) {
			t.Fatalf("confined path %q not under root %q", got, root)
		}
	}
}

func TestAssertPathUnderAnyRootRejectsHostSecrets(t *testing.T) {
	root := t.TempDir()
	secrets := "/var/run/secrets/kubernetes.io/serviceaccount"
	if _, err := AssertPathUnderAnyRoot(secrets, []string{root, "/tmp"}); err == nil {
		t.Fatal("expected rejection of host secrets path")
	}
}

func TestAssertPathUnderAnyRootRejectsEscape(t *testing.T) {
	root := t.TempDir()
	escape := filepath.Join(root, "..", "..", "etc")
	if _, err := AssertPathUnderAnyRoot(escape, []string{root}); err == nil {
		t.Fatal("expected rejection of path escaping root")
	}
}

func TestAssertPathUnderAnyRootRequiresRoots(t *testing.T) {
	if _, err := AssertPathUnderAnyRoot("/tmp/x", nil); err == nil {
		t.Fatal("expected error when no roots configured")
	}
}

func isUnder(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if path == root {
		return true
	}
	return len(path) > len(root) && path[:len(root)+1] == root+string(filepath.Separator)
}
