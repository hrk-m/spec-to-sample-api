#!/bin/sh
set -eu

MYSQL_HOST="${MYSQL_HOST:-localhost}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-user}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-password}"
MYSQL_DATABASE="${MYSQL_DATABASE:-sample}"

DB_URL="mysql://${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}"

echo "Running migrations..."
MIGRATION_OUTPUT=""
if ! MIGRATION_OUTPUT="$(migrate -path /app/db/migrate -database "$DB_URL" up 2>&1)"; then
  printf '%s\n' "$MIGRATION_OUTPUT"
  if ! printf '%s' "$MIGRATION_OUTPUT" | grep -q "no change"; then
    exit 1
  fi
else
  printf '%s\n' "$MIGRATION_OUTPUT"
fi

echo "Applying seed data..."
for f in /app/db/seed/*.sql; do
  echo "  Applying $f"
  mysql --ssl=0 --default-auth=mysql_native_password -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" < "$f"
done

echo "Starting API server..."
exec /app/api
