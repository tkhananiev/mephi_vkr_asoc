# Веб-клиент (`web/`)

Приложение на **React + TypeScript + Vite**. Запросы к `/api/v1/...` в dev-режиме проксируются на локальные порты сервисов см. **`vite.config.ts`** (обход CORS: `8080` api/scans, `8081` sync, `8082`/`8083` — processing/jira).

**Страницы:** дашборд (`/`), запуск Semgrep (`/scan`), ручной синхрон справочников (`/reference`), доска групп уязвимостей (`/groups`).

**UI:** тёмный «command center» (Syne + IBM Plex, градиенты, сетка на фоне, карточки действий на синхроне, «окно» с ответом JSON на скане, бейджи severity на группах). Это **живой макет** в коде, не отдельный Figma.

## Запуск в разработке

Из корня `mephi_vkr_asoc` (после `docker compose up` backend):

```bash
cd web
npm install
npm run dev
```

Откройте URL, который выведет Vite (по умолчанию `http://localhost:5173`). Сначала должны быть запущены сервисы из `deploy/docker-compose.yml` на портах, совпадающих с прокси.

## Сборка

```bash
npm run build
```

Артефакты — в `web/dist/`. Подробности архитектуры backend и потоков данных: [`../docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md).
