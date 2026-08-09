package postgres

import (
	"strings"
	"testing"
)

func TestSQLGroupVisibleForProductUsesKeyPrefix(t *testing.T) {
	sql := sqlGroupVisibleForProduct("$2", "$3")
	if !strings.Contains(sql, `g.group_key LIKE ('u:' || $2::bigint::text || ':p:' || $3::bigint::text || ':%')`) {
		t.Fatalf("missing owner+product key prefix match: %s", sql)
	}
	if !strings.Contains(sql, `g.group_key LIKE ('p:' || $3::bigint::text || ':%')`) {
		t.Fatalf("missing product-only key prefix match: %s", sql)
	}
	if !strings.Contains(sql, `g.group_key NOT LIKE ('u:' || $2::bigint::text || ':p:%')`) {
		t.Fatalf("missing legacy exclusion of product-scoped keys: %s", sql)
	}
	if strings.Contains(sql, `%%`) {
		t.Fatalf("fmt escaping leaked into SQL: %s", sql)
	}
}
