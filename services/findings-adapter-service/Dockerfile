FROM golang:1.23 AS builder

WORKDIR /src
COPY services/findings-adapter-service/go.mod ./
RUN go mod download
COPY services/findings-adapter-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/findings-adapter-service ./cmd/findings-adapter-service

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
COPY --from=builder /out/findings-adapter-service /usr/local/bin/findings-adapter-service

ENV APP_HTTP_PORT=8090
EXPOSE 8090
CMD ["findings-adapter-service"]
