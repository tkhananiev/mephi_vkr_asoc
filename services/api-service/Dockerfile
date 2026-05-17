FROM golang:1.23 AS builder

WORKDIR /src
COPY services/api-service/go.mod ./
RUN go mod download
COPY services/api-service ./
# -p 1 стабильнее при сборке образа linux/amd64 на хостах с QEMU/Rosetta (иногда падал toolchain с SIGSEGV).
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -p 1 -o /out/api-service ./cmd/api-service

FROM alpine:3.20

RUN apk --no-cache add ca-certificates docker-cli
COPY --from=builder /out/api-service /usr/local/bin/api-service

EXPOSE 8080
CMD ["api-service"]
