#!/usr/bin/env bash
# scripts/migrate.sh
#
# Applies (or reverts) the main database migrations using the `migrate`
# CLI, reading the connection string from .env.
#
# Usage:
#   ./scripts/migrate.sh up          # applies all pending migrations
#   ./scripts/migrate.sh down 1      # reverts the last applied migration
#   ./scripts/migrate.sh version     # shows the current schema version
#   ./scripts/migrate.sh force 1     # forces the version (advanced use, be careful)
set -euo pipefail
 
cd "$(dirname "$0")/.."
 
if [ ! -f .env ]; then
  echo "Error: .env file not found. Copy .env.example to .env first." >&2
  exit 1
fi
 
set -a
source .env
set +a
 
: "${POSTGRES_USER:?POSTGRES_USER is not set in .env}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is not set in .env}"
: "${POSTGRES_DB:?POSTGRES_DB is not set in .env}"
: "${POSTGRES_DB_PORT:?POSTGRES_DB_PORT is not set in .env}"
 
DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_DB_PORT}/${POSTGRES_DB}?sslmode=disable"
 
if ! command -v migrate >/dev/null 2>&1; then
  echo "Error: the 'migrate' CLI is not installed." >&2
  echo "Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" >&2
  echo "And make sure \$(go env GOPATH)/bin is in your PATH." >&2
  exit 1
fi
 
echo "==> Using: ${DATABASE_URL}"
echo "==> Command: migrate $*"
 
migrate -path=./migrations -database "${DATABASE_URL}" "$@"
 
echo "==> Done."