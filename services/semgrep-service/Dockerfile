FROM golang:1.23 AS builder

WORKDIR /src
COPY services/semgrep-service/go.mod ./
RUN go mod download
COPY services/semgrep-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/semgrep-service ./cmd/semgrep-service

FROM python:3.11-slim

WORKDIR /app
RUN pip install --no-cache-dir semgrep
COPY --from=builder /out/semgrep-service /usr/local/bin/semgrep-service
COPY demo /app/demo

EXPOSE 8085
CMD ["semgrep-service"]
