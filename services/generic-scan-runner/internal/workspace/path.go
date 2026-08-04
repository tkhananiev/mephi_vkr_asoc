package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AssertPathUnderAnyRoot ensures path (after symlink resolution) stays under
// at least one allowed root. Roots are typically APP_ALLOWED_SCAN_ROOTS.
func AssertPathUnderAnyRoot(path string, allowedRoots []string) (string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return "", fmt.Errorf("empty target path")
	}
	var roots []string
	for _, root := range allowedRoots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || root == "." {
			continue
		}
		roots = append(roots, root)
	}
	if len(roots) == 0 {
		return "", fmt.Errorf("allowed scan root is not configured")
	}

	var lastErr error
	for _, root := range roots {
		confined, err := assertInsideRoot(root, path)
		if err == nil {
			return confined, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("target path %q is outside allowed scan roots", path)
	}
	return "", fmt.Errorf("target path %q is outside allowed scan roots: %w", path, lastErr)
}

func assertInsideRoot(root, path string) (string, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolvedRoot = filepath.Clean(root)
	}
	rootSep := resolvedRoot + string(filepath.Separator)
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		cleaned := filepath.Clean(path)
		if cleaned != resolvedRoot && !strings.HasPrefix(cleaned, rootSep) {
			return "", fmt.Errorf("path escapes allowed root %q", resolvedRoot)
		}
		return cleaned, nil
	}
	if resolvedPath != resolvedRoot && !strings.HasPrefix(resolvedPath, rootSep) {
		return "", fmt.Errorf("path escapes allowed root %q", resolvedRoot)
	}
	return resolvedPath, nil
}
