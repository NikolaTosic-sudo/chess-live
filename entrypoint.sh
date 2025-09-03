#!/bin/sh
set -e

echo "Waiting for Postgres to be ready..."
until pg_isready -h $(echo $DB_URL | sed -E 's|.*@([^:]+):.*|\1|') -p 5432; do
  sleep 1
done

echo "Running database migrations..."
goose -dir ./sql/schema postgres "$DB_URL" up

echo "Starting chess-live on port $PORT..."
exec /usr/bin/chess-live
