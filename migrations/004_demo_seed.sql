INSERT INTO catalog.reference_records (
    source_code, external_id, title, description, severity, source_url, status, metadata_json
)
VALUES
(
    'nvd',
    'CVE-2021-44228',
    'CVE-2021-44228',
    'Demo seed record for end-to-end vulnerability management scenario.',
    'high',
    'https://nvd.nist.gov/vuln/detail/CVE-2021-44228',
    'published',
    '{"seed": true, "source": "demo"}'::jsonb
),
(
    'bdu_fstec',
    'BDU:2021-00001',
    'Демонстрационная запись БДУ ФСТЭК',
    'Seed record for demo correlation with CVE-2021-44228.',
    'high',
    'https://bdu.fstec.ru/',
    'published',
    '{"seed": true, "source": "demo"}'::jsonb
)
ON CONFLICT (source_code, external_id) DO NOTHING;

INSERT INTO catalog.reference_aliases (reference_record_id, alias_type, alias_value)
SELECT id, 'CVE', 'CVE-2021-44228'
FROM catalog.reference_records
WHERE source_code = 'nvd' AND external_id = 'CVE-2021-44228'
ON CONFLICT DO NOTHING;

INSERT INTO catalog.reference_aliases (reference_record_id, alias_type, alias_value)
SELECT id, 'CVE', 'CVE-2021-44228'
FROM catalog.reference_records
WHERE source_code = 'bdu_fstec' AND external_id = 'BDU:2021-00001'
ON CONFLICT DO NOTHING;

INSERT INTO catalog.reference_aliases (reference_record_id, alias_type, alias_value)
SELECT id, 'BDU', 'BDU:2021-00001'
FROM catalog.reference_records
WHERE source_code = 'bdu_fstec' AND external_id = 'BDU:2021-00001'
ON CONFLICT DO NOTHING;
