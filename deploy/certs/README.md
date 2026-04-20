# Сертификаты для TLS к фиду БДУ ФСТЭК

Файлы **Russian Trusted** (корневой + выпускающий УЦ Минцифры), нормализованные PEM. Источник при сборке каталога: публичные URL `https://gu-st.ru/content/lending/russian_trusted_*_ca_pem.crt` (тот же набор, что раздаётся с портала Госуслуг).

- `russian_trusted_chain.pem` — корень + sub в одном файле (использует `reference-data-service` через `APP_BDU_ROOT_CA_FILE`).
- Отдельные файлы `russian_trusted_root.pem`, `russian_trusted_sub.pem` — для ручной проверки или обновления.

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
