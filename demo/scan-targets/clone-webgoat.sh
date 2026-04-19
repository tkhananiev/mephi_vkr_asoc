#!/usr/bin/env sh
# Клонирует OWASP WebGoat в этот каталог (на хосте). В compose путь в контейнере semgrep: /app/demo/scan-targets/WebGoat
set -e
cd "$(dirname "$0")"
if [ -d WebGoat ]; then
  echo "WebGoat: каталог уже есть, пропуск клонирования."
  exit 0
fi
git clone --depth 1 https://github.com/WebGoat/WebGoat.git
