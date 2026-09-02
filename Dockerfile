FROM golang:1.25-alpine AS api-build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/permission-api ./cmd/api-server

FROM alpine:3.22 AS api

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S permission-center \
    && adduser -S -G permission-center permission-center
WORKDIR /app
COPY --from=api-build /out/permission-api /app/permission-api
COPY configs /app/configs
USER permission-center
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=6 \
    CMD wget --quiet --spider http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/permission-api"]

FROM postgres:16-alpine AS migrator

WORKDIR /app
COPY migrations /app/migrations
COPY scripts/migrate.sh /app/scripts/migrate.sh
RUN chmod +x /app/scripts/migrate.sh
ENTRYPOINT ["/app/scripts/migrate.sh"]
