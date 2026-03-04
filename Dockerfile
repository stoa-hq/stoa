FROM node:20-alpine AS frontend

WORKDIR /build

# Admin Frontend
COPY admin/package.json admin/package-lock.json* admin/
RUN cd admin && npm ci

COPY admin/ admin/
RUN cd admin && npm run build

# Storefront
COPY storefront/package.json storefront/package-lock.json* storefront/
RUN cd storefront && npm ci

COPY storefront/ storefront/
RUN cd storefront && npm run build

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Eingebaute Frontends aus dem Frontend-Stage übernehmen
COPY --from=frontend /build/internal/admin/build ./internal/admin/build
COPY --from=frontend /build/internal/storefront/build ./internal/storefront/build

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o stoa ./cmd/stoa

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/stoa .
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/config.example.yaml ./config.yaml

EXPOSE 8080

ENTRYPOINT ["./stoa"]
CMD ["serve"]
