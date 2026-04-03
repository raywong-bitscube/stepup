FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY mock-ai ./mock-ai

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mock-ai ./mock-ai/main.go

FROM alpine:3.20

WORKDIR /srv
COPY --from=builder /out/mock-ai /srv/mock-ai

EXPOSE 8090
ENTRYPOINT ["/srv/mock-ai"]
