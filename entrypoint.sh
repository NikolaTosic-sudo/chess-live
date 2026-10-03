#!/bin/sh
set -e

: "${DB_URL:?DB_URL must be set}"
export PORT="${PORT:-8080}"

echo "Waiting for Postgres to be ready..."

until pg_isready -d "$DB_URL"; do
  sleep 1
done

echo "Running database migrations..."
goose -dir ./sql/schema postgres "$DB_URL" up

echo "Starting chess-live on port $PORT..."
exec /usr/bin/chess-live
