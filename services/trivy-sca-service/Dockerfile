FROM golang:1.23 AS builder

WORKDIR /src
COPY services/trivy-sca-service/go.mod ./
RUN go mod download
COPY services/trivy-sca-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/trivy-sca-service ./cmd/trivy-sca-service

FROM debian:bookworm-slim

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates curl \
  && rm -rf /var/lib/apt/lists/*

ARG TRIVY_VERSION=0.70.0
RUN curl -fsSL -L "https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_Linux-64bit.tar.gz" \
  | tar -xzf - -C /usr/local/bin trivy \
  && chmod +x /usr/local/bin/trivy

COPY --from=builder /out/trivy-sca-service /usr/local/bin/trivy-sca-service
RUN mkdir -p /app/demo/scan-targets \
  && cd /app/demo/scan-targets \
  && GIT_TERMINAL_PROMPT=0 git clone --depth 1 https://github.com/WebGoat/WebGoat.git \
  && test -d WebGoat

ENV APP_HTTP_PORT=8088
EXPOSE 8088
CMD ["trivy-sca-service"]
