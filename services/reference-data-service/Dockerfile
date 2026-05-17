FROM golang:1.23 AS builder

WORKDIR /src
COPY services/reference-data-service/go.mod services/reference-data-service/go.sum ./
RUN go mod download
COPY services/reference-data-service ./

# На слабых VPS компилятор Go может получить SIGKILL (OOM); один поток снижает пик RAM.
ENV GOMAXPROCS=1
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/reference-data-service ./cmd/reference-data-service

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
COPY --from=builder /out/reference-data-service /usr/local/bin/reference-data-service

# Полный дамп БДУ (обновляются в deploy/embed/bdu-bundled/ перед сборкой при необходимости).
COPY deploy/embed/bdu-bundled/vulxml.zip /opt/asoc/bdu-default/vulxml.zip
COPY deploy/embed/bdu-bundled/vullist.xlsx /opt/asoc/bdu-default/vullist.xlsx

EXPOSE 8081
CMD ["reference-data-service"]
