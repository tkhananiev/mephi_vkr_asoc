# Demo Guide

Сервис справочников: **`reference-data-service`**, порт **8081** (в `deploy/docker-compose.yml`). Подставь вместо **`localhost`** свой хост, если вызываешь не с той же машины (например `45.87.246.170`).

Во всех примерах ниже: метод **POST**, тела **нет** (в Postman: Body → none).

---

## 1. Синхронизация баз БДУ и CVE (NVD)

**CVE** в системе приходят из **NVD** (REST API 2.0). **БДУ ФСТЭК** — отдельный источник (RSS-фид).

### 1.1. БДУ ФСТЭК

| | |
|--|--|
| **Метод** | `POST` |
| **URL** | `http://localhost:8081/api/v1/sync/bdu` |

**curl:**

```bash
curl -X POST "http://localhost:8081/api/v1/sync/bdu"
```

**Postman:** New Request → POST → вставить URL → Send.

**Ответ при успехе:** `202 Accepted`, JSON с полями `source_code`, `run_id`, `items_discovered`, `items_processed`, …

Если фид недоступен, сервис может отработать с **демонстрационной записью** (см. код адаптера БДУ).

---

### 1.2. NVD (CVE) — полная выгрузка каталога

Запрос **без** `cve_id` обходит **все страницы** ответа NVD API 2.0 (до **2000** CVE на страницу, пока не исчерпан `totalResults`). Между запросами действует пауза по лимитам NVD (~5 запросов / 30 с без ключа; с ключом — быстрее).

Переменные окружения `reference-data-service` (см. `docs/ENVIRONMENT.md`):

- **`APP_NVD_API_KEY`** — необязательно; [ключ NVD](https://nvd.nist.gov/developers/request-an-api-key) в заголовке `apiKey`, выше лимит и короче полная синхронизация.
- **`APP_NVD_PAGE_SIZE`** — размер страницы (по умолчанию **2000**, максимум NVD).
- **`APP_NVD_MAX_PAGES`** — ограничить число страниц за один прогон (**0** = без ограничения). Для теста можно поставить `1`.

Запрос может идти **десятки минут** и держит открытым HTTP-соединение; в Postman увеличь **timeout** (Settings → General).

| | |
|--|--|
| **Метод** | `POST` |
| **URL** | `http://localhost:8081/api/v1/sync/nvd` |

**curl** (долгий запрос):

```bash
curl --max-time 0 -X POST "http://localhost:8081/api/v1/sync/nvd"
```

**Postman:** POST → URL как выше → Send (увеличить таймаут запроса).

---

### 1.3. NVD (CVE) — один конкретный идентификатор

| | |
|--|--|
| **Метод** | `POST` |
| **URL** | `http://localhost:8081/api/v1/sync/nvd?cve_id=CVE-2021-44228` |

**curl:**

```bash
curl -X POST "http://localhost:8081/api/v1/sync/nvd?cve_id=CVE-2021-44228"
```

**Postman:** тот же URL в строке запроса; вкладка **Params**: ключ `cve_id`, значение `CVE-2021-44228` (или другой CVE).

---

### 1.4. БДУ и NVD подряд (одним запросом)

| | |
|--|--|
| **Метод** | `POST` |
| **URL** | `http://localhost:8081/api/v1/sync/all` |

**curl:**

```bash
curl -X POST "http://localhost:8081/api/v1/sync/all"
```

Полная синхронизация NVD может занять много времени; при необходимости вызывай **§1.1** и **§1.2** по отдельности вместо `sync/all`.

---

### 1.5. История прогонов синхронизации

| | |
|--|--|
| **Метод** | `GET` |
| **URL** | `http://localhost:8081/api/v1/sync/runs?limit=10` |

**curl:**

```bash
curl "http://localhost:8081/api/v1/sync/runs?limit=10"
```

**Postman:** GET → тот же URL.
