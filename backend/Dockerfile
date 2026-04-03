FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY backend ./backend

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./backend/cmd/server

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
WORKDIR /srv

COPY --from=builder /out/server /srv/server
COPY --chown=app:app frontend-admin /srv/static/admin
COPY --chown=app:app frontend-student /srv/static/student

ENV STATIC_DIR=/srv/static

USER app
EXPOSE 8080

ENTRYPOINT ["/srv/server"]
