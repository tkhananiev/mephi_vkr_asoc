-- Демо: корреляция находок Semgrep со справочником по CWE (без CVE в правиле).
INSERT INTO catalog.reference_aliases (reference_record_id, alias_type, alias_value)
SELECT id, 'CWE', 'CWE-78'
FROM catalog.reference_records
WHERE source_code = 'nvd' AND external_id = 'CVE-2021-44228'
ON CONFLICT DO NOTHING;
