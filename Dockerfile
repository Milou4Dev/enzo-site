FROM golang:1.23.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main .

FROM alpine:3.20.3
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D appuser && \
    mkdir -p /app/templates /app/static && \
    chown -R appuser:appuser /app

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates/
COPY --from=builder /app/static ./static/

USER appuser
ENV PORT=8080 \
    GIN_MODE=release \
    TZ=UTC

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

CMD ["./main"]