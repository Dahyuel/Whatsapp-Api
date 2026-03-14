# ── Build Stage ─────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /whatsapp-api ./cmd/server

# ── Runtime Stage ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /whatsapp-api /app/whatsapp-api

# Create required directory structure
RUN mkdir -p /app/data/sessions /app/data/media

EXPOSE 3000

VOLUME ["/app/data"]

ENTRYPOINT ["/app/whatsapp-api"]
