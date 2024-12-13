FROM golang:1.23.4-alpine AS builder
RUN apk add --no-cache git ca-certificates build-base
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s -extldflags '-static' -X 'main.Version=$(git describe --tags --always)' -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o main .
RUN ./main -version || true

FROM aquasec/trivy:latest AS security-scan
COPY --from=builder /build /build
RUN trivy filesystem --no-progress --ignore-unfixed --severity HIGH,CRITICAL /build

FROM alpine:3.20.3 AS runtime
RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates 2>/dev/null || true
RUN adduser -D -u 10001 appuser && mkdir -p /app/templates /app/static && chown -R appuser:appuser /app
WORKDIR /app
COPY --from=builder --chown=appuser:appuser /build/main .
COPY --from=builder --chown=appuser:appuser /build/templates ./templates/
COPY --from=builder --chown=appuser:appuser /build/static ./static/
USER appuser
ENV PORT=8080 GIN_MODE=release TZ=UTC GOMAXPROCS=4
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -q --spider http://localhost:8080/health || exit 1
ENV GOGC=100 GOMEMLIMIT=256MiB
ENTRYPOINT ["./main"]