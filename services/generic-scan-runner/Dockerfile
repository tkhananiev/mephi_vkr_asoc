FROM golang:1.23 AS builder

WORKDIR /src
COPY services/generic-scan-runner/go.mod ./
RUN go mod download
COPY services/generic-scan-runner ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -p 1 -o /out/generic-scan-runner ./cmd/generic-scan-runner

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/generic-scan-runner /usr/local/bin/generic-scan-runner

ENV APP_HTTP_PORT=8087
EXPOSE 8087
CMD ["generic-scan-runner"]
