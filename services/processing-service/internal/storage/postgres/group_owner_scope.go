package postgres

// Owner-scoped group access must key off group_key (u:<owner>:...), not shared
// vulnerability rows. core.vulnerabilities is unique by CVE/product/version/CWE
// globally, so multiple tenants' groups link to the same vulnerability id.

const updateGroupStatusSQL = `
		UPDATE core.vulnerability_groups g
		SET status = $2, updated_at = NOW()
		WHERE g.id = $1
		  AND (
			$3::bigint IS NULL
			OR g.group_key LIKE ('u:' || $3::bigint::text || ':%')
		  )
		RETURNING g.id, g.group_key, g.grouping_rule, g.severity_max, g.assets_count, g.status
`

// listVulnerabilityReportOwnerGroupPredicate is the tenant gate for report rows.
// It must appear in ListVulnerabilityReport in addition to finding-owner EXISTS
// checks, otherwise another user's group_key is returned for a shared CVE row.
const listVulnerabilityReportOwnerGroupPredicate = `g.group_key LIKE ('u:' || $2::bigint::text || ':%')`
