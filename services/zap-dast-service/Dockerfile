FROM golang:1.23 AS builder

WORKDIR /src
COPY services/zap-dast-service/go.mod ./
RUN go mod download
COPY services/zap-dast-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/zap-dast-service ./cmd/zap-dast-service

FROM ghcr.io/zaproxy/zaproxy:stable

USER root
COPY --from=builder /out/zap-dast-service /usr/local/bin/zap-dast-service

ENV APP_HTTP_PORT=8089
ENV APP_ZAP_HOME=/zap
ENV APP_ZAP_SCAN_TIMEOUT_MIN=8

USER zap
WORKDIR /zap
EXPOSE 8089
CMD ["zap-dast-service"]
