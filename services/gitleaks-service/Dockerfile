FROM golang:1.23 AS builder

WORKDIR /src
COPY services/gitleaks-service/go.mod ./
RUN go mod download
COPY services/gitleaks-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/gitleaks-service ./cmd/gitleaks-service

FROM debian:bookworm-slim

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates curl \
  && rm -rf /var/lib/apt/lists/*

ARG GITLEAKS_VERSION=8.21.2
RUN curl -fsSL "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" \
  | tar -xz -C /usr/local/bin gitleaks \
  && chmod +x /usr/local/bin/gitleaks

COPY --from=builder /out/gitleaks-service /usr/local/bin/gitleaks-service
COPY demo /app/demo
RUN cd /app/demo/scan-targets && GIT_TERMINAL_PROMPT=0 git clone --depth 1 https://github.com/WebGoat/WebGoat.git \
  && test -d WebGoat

ENV APP_HTTP_PORT=8086
EXPOSE 8086
CMD ["gitleaks-service"]
