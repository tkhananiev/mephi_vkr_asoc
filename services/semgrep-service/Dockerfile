FROM golang:1.23 AS builder

WORKDIR /src
COPY services/semgrep-service/go.mod ./
RUN go mod download
COPY services/semgrep-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/semgrep-service ./cmd/semgrep-service

FROM python:3.11-slim

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends git \
  && rm -rf /var/lib/apt/lists/* \
  && pip install --no-cache-dir semgrep
COPY --from=builder /out/semgrep-service /usr/local/bin/semgrep-service
COPY demo /app/demo
# WebGoat не хранится в git (см. demo/scan-targets/README.md); для образа клонируем при сборке.
RUN cd /app/demo/scan-targets && GIT_TERMINAL_PROMPT=0 git clone --depth 1 https://github.com/WebGoat/WebGoat.git \
  && test -d WebGoat

EXPOSE 8085
CMD ["semgrep-service"]
