FROM golang:1.23 AS builder

WORKDIR /src
COPY services/processing-service/go.mod services/processing-service/go.sum ./
RUN go mod download
COPY services/processing-service ./

ENV GOMAXPROCS=1
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/processing-service ./cmd/processing-service

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
COPY --from=builder /out/processing-service /usr/local/bin/processing-service

EXPOSE 8082
CMD ["processing-service"]
