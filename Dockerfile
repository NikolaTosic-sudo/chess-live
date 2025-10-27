# ─────────────────────────────
# 1. Build stage
# ─────────────────────────────
FROM golang:1.25 AS builder

# Install goose + templ
RUN go install github.com/pressly/goose/v3/cmd/goose@latest \
	&& mkdir -p /app/bin \
	&& cp $(go env GOPATH)/bin/goose /app/bin/goose \
	&& go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Copy dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate templ files if needed
RUN templ generate

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o chess-live .

# ─────────────────────────────
# 2. Runtime stage
# ─────────────────────────────
FROM postgres:15 AS runner

RUN apt-get update --fix-missing \
	&& apt-get install -y --no-install-recommends ca-certificates curl postgresql-client \
	&& curl -L https://github.com/pressly/goose/releases/latest/download/goose_linux_amd64 -o /usr/bin/goose \
	&& chmod +x /usr/bin/goose \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary
COPY --from=builder /app/chess-live /usr/bin/chess-live
COPY --from=builder /app/bin/goose /usr/bin/goose

# Copy static assets & schema
COPY assets/ ./assets/
COPY sql/schema ./sql/schema

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
