# 1. Build stage
FROM golang:1.26 AS builder

WORKDIR /app

# Build a static Goose binary for the runtime stage
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.28.0 \
  && mkdir -p /app/bin \
  && cp "$(go env GOPATH)/bin/goose" /app/bin/goose

# Copy dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Match the templ generator to the project's runtime library
RUN go install github.com/a-h/templ/cmd/templ@$(go list -m -f '{{.Version}}' github.com/a-h/templ)

# Copy source code and generate templates
COPY . .
RUN templ generate

# Build application binary
RUN CGO_ENABLED=0 GOOS=linux go build -o chess-live .

# 2. Runtime stage
FROM postgres:15 AS runner

RUN apt-get update --fix-missing \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy application and migration binaries
COPY --from=builder /app/chess-live /usr/bin/chess-live
COPY --from=builder /app/bin/goose /usr/bin/goose

# Copy static assets and migrations
COPY assets/ ./assets/
COPY sql/schema/ ./sql/schema/

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
