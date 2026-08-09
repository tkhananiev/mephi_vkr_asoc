package postgres

import "fmt"

// sqlGroupVisibleForProduct returns a SQL boolean expression over vulnerability_groups
// alias `g` that enforces console product isolation.
//
// Product-scoped group_keys (u:<owner>:p:<product>:… or p:<product>:…) match by
// key prefix so sibling products that share a global vulnerabilities row cannot
// list/mutate each other's groups via finding→vuln EXISTS joins.
//
// Legacy keys without a :p: segment keep the finding→run product EXISTS fallback.
func sqlGroupVisibleForProduct(ownerParam, productParam string) string {
	return fmt.Sprintf(`(
			%s::bigint IS NULL
			OR g.group_key LIKE ('u:' || %s::bigint::text || ':p:' || %s::bigint::text || ':%%')
			OR g.group_key LIKE ('p:' || %s::bigint::text || ':%%')
			OR (
				g.group_key NOT LIKE ('u:' || %s::bigint::text || ':p:%%')
				AND g.group_key NOT LIKE 'p:%%'
				AND EXISTS (
					SELECT 1
					FROM core.group_vulnerabilities gv_ps
					JOIN core.finding_vulnerabilities fv_ps ON fv_ps.vulnerability_id = gv_ps.vulnerability_id
					JOIN core.findings f_ps ON f_ps.id = fv_ps.finding_id
					JOIN core.processing_runs pr_ps ON pr_ps.id = f_ps.processing_run_id
					WHERE gv_ps.group_id = g.id
					  AND (%s::bigint IS NULL OR pr_ps.owner_user_id = %s)
					  AND pr_ps.console_product_id = %s
				)
			)
		)`, productParam, ownerParam, productParam, productParam, ownerParam, ownerParam, ownerParam, productParam)
}
