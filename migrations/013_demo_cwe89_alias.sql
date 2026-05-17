-- Демо: корреляция находок SAST по CWE-89 (SQLi) с тестовой записью NVD из 004_demo_seed;
-- для отчёта подставляются CVE и БДУ по связям seed (учебный сценарий, не соответствие реальной CVE).
INSERT INTO catalog.reference_aliases (reference_record_id, alias_type, alias_value)
SELECT id, 'CWE', 'CWE-89'
FROM catalog.reference_records
WHERE source_code = 'nvd' AND external_id = 'CVE-2021-44228'
ON CONFLICT DO NOTHING;
