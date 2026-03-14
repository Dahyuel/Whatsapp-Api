# ── Build Stage ─────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod tidy
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /whatsapp-api ./cmd/server

# ── Runtime Stage ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata sqlite

WORKDIR /app

COPY --from=builder /whatsapp-api /app/whatsapp-api

RUN mkdir -p /app/data/sessions /app/data/media

EXPOSE 3000
VOLUME ["/app/data"]

ENTRYPOINT ["/app/whatsapp-api"]