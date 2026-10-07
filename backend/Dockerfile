FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/ticket-contract-backfill ./cmd/ticket-contract-backfill

FROM debian:bookworm-slim
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates tzdata wkhtmltopdf fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/server /app/ticket-contract-backfill ./
COPY templates/ ./templates/
EXPOSE 8080
CMD ["/app/server"]
