# Сертификаты Russian Trusted (опционально)

Файлы **Russian Trusted** (корневой + выпускающий УЦ Минцифры), нормализованные PEM. Источник при сборке: `https://gu-st.ru/content/lending/russian_trusted_*_ca_pem.crt`.

**В compose по умолчанию** для БДУ включён `APP_BDU_INSECURE_SKIP_VERIFY=true`: у ФСТЭК подпись листа может идти промежуточным сертификатом **другой** выдачи, чем в этой цепочке, из‑за чего OpenSSL даёт `authority and subject key identifier mismatch`. Плюс часть площадок отдаёт **403** клиентам без браузерного `User-Agent` (в коде БДУ-клиента задан обычный Chrome UA).

- `russian_trusted_chain.pem` — для ручного `curl --cacert` или эксперимента с `APP_BDU_ROOT_CA_FILE` после обновления sub с [CDP НУЦ](https://nuc-cdp.digital.gov.ru/) под актуальный лист.
- Отдельные `russian_trusted_root.pem`, `russian_trusted_sub.pem` — см. ниже обновление.

## Обновление цепочки (на машине с доступом в интернет)

```bash
cd "$(dirname "$0")"
curl -fsSL 'https://gu-st.ru/content/lending/russian_trusted_root_ca_pem.crt' | tr -d '\r' > /tmp/rt-root.pem
curl -fsSL 'https://gu-st.ru/content/lending/russian_trusted_sub_ca_pem.crt' | tr -d '\r' > /tmp/rt-sub.pem
openssl x509 -in /tmp/rt-root.pem -out russian_trusted_root.pem
openssl x509 -in /tmp/rt-sub.pem -out russian_trusted_sub.pem
cat russian_trusted_root.pem russian_trusted_sub.pem > russian_trusted_chain.pem
```

Если УЦ ротируют, после обновления пересоберите/перезапустите `reference-data-service`.
