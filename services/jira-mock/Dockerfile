FROM golang:1.23 AS builder

WORKDIR /src
COPY services/jira-mock/go.mod ./
RUN go mod download
COPY services/jira-mock ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/jira-mock ./cmd/jira-mock

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
COPY --from=builder /out/jira-mock /usr/local/bin/jira-mock

EXPOSE 8090
CMD ["jira-mock"]
