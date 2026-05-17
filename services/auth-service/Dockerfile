FROM golang:1.23 AS builder

WORKDIR /src
COPY services/auth-service/go.mod ./
RUN go mod download
COPY services/auth-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auth-service ./cmd/auth-service

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
COPY --from=builder /out/auth-service /usr/local/bin/auth-service

EXPOSE 8091
CMD ["auth-service"]
