FROM golang:1.23 AS builder

WORKDIR /src
COPY services/semgrep-service/go.mod ./
RUN go mod download
COPY services/semgrep-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/semgrep-service ./cmd/semgrep-service

FROM semgrep/semgrep:1.97.0

WORKDIR /app
COPY --from=builder /out/semgrep-service /usr/local/bin/semgrep-service
RUN mkdir -p /app/demo/scan-targets \
  && cd /app/demo/scan-targets \
  && GIT_TERMINAL_PROMPT=0 git clone --depth 1 https://github.com/WebGoat/WebGoat.git \
  && test -d WebGoat

EXPOSE 8085
CMD ["semgrep-service"]
