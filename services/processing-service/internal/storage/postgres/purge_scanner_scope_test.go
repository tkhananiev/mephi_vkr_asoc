package postgres

import (
	"regexp"
	"strconv"
	"testing"
)

func TestPurgeScannerScopeStatementsBindAllPlaceholders(t *testing.T) {
	ownerID := int64(42)
	consoleProductID := int64(7)

	for i, stmt := range purgeScannerScopeStatements("semgrep", &ownerID, &consoleProductID) {
		maxPlaceholder, seen := queryPlaceholders(stmt.query)
		if maxPlaceholder != len(stmt.args) {
			t.Fatalf("statement %d uses placeholders through $%d with %d args", i, maxPlaceholder, len(stmt.args))
		}
		for n := 1; n <= maxPlaceholder; n++ {
			if !seen[n] {
				t.Fatalf("statement %d skips placeholder $%d", i, n)
			}
		}
	}
}

var postgresPlaceholderRe = regexp.MustCompile(`\$(\d+)`)

func queryPlaceholders(query string) (int, map[int]bool) {
	seen := make(map[int]bool)
	max := 0
	for _, match := range postgresPlaceholderRe.FindAllStringSubmatch(query, -1) {
		n, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		seen[n] = true
		if n > max {
			max = n
		}
	}
	return max, seen
}
