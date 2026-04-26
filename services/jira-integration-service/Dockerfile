FROM golang:1.23 AS builder

WORKDIR /src
COPY services/jira-integration-service/go.mod services/jira-integration-service/go.sum ./
RUN go mod download
COPY services/jira-integration-service ./

ENV GOMAXPROCS=1
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/jira-integration-service ./cmd/jira-integration-service

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
COPY --from=builder /out/jira-integration-service /usr/local/bin/jira-integration-service

EXPOSE 8083
CMD ["jira-integration-service"]
