#!/bin/sh
# Starts the backend stack (MongoDB, SeaweedFS, API) with Docker Compose,
# seeds a QA admin, and runs the API smoke test against it. Used by CI;
# runs the same way locally (it reuses a stack that's already up).
set -eu
cd "$(dirname "$0")/../.."

API="${QA_API:-http://localhost:8080}"
ADMIN_EMAIL="${QA_ADMIN_EMAIL:-qa-admin@betailo.test}"
ADMIN_PASSWORD="${QA_ADMIN_PASSWORD:-Qa-admin-smoke-2026}"

docker compose up -d --build mongo seaweedfs backend

echo "waiting for the API…"
for _ in $(seq 1 60); do
  curl -sf "$API/health" >/dev/null && break
  sleep 2
done
curl -sf "$API/health" >/dev/null || { docker compose logs backend; exit 1; }

# The API never lets anyone register as admin: register, then promote.
curl -s -o /dev/null -X POST "$API/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\",\"role\":\"buyer\",\"displayName\":\"QA admin\"}"
docker compose exec -T mongo mongosh --quiet betailo \
  --eval "db.users.updateOne({email: '$ADMIN_EMAIL'}, {\$set: {role: 'admin', suspended: false}})" >/dev/null

QA_API="$API" QA_ADMIN_EMAIL="$ADMIN_EMAIL" QA_ADMIN_PASSWORD="$ADMIN_PASSWORD" \
  python3 scripts/qa/api_smoke.py
