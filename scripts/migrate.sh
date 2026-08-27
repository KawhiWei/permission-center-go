#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
DATABASE_URL=${PERMISSION_CENTER_DATABASE_URL:-postgres://postgres:postgres@localhost:5432/permission_center?sslmode=disable}
for migration in "$ROOT"/migrations/*.sql; do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
done
