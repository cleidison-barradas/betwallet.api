#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

cleanup() {
  echo "==> Tearing down containers..."
  docker compose down -v
}
trap cleanup EXIT

echo "==> Starting Postgres..."
docker compose up -d postgres
docker compose ps postgres

echo "==> Waiting for Postgres to be ready..."
until docker compose exec -T postgres pg_isready -U root >/dev/null 2>&1; do
  sleep 1
done

echo "==> Creating test database..."
docker compose exec -T postgres createdb -U root betwallet_test 2>/dev/null || true

echo "==> Recreating schema (clean state on every run)..."
docker compose exec -T postgres psql -U root -d betwallet_test \
  -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

echo "==> Applying migrations..."
for f in migrations/*.up.sql; do
  echo "  - $f"
  docker compose exec -T postgres psql -U root -d betwallet_test -f - < "$f"
done

echo "==> Running integration tests..."
DATABASE_URL="postgres://root:root@localhost:5432/betwallet_test?sslmode=disable" \
  go test ./internal/app -v -tags=integration -run Integration -v