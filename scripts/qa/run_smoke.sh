#!/bin/sh
# Starts the backend stack (MongoDB, SeaweedFS, API) with Docker Compose,
# seeds a QA admin, and runs the API smoke test against it. Used by CI;
# runs the same way locally (it reuses a stack that's already up).
#
#   QA_RESET=1   drop the local database first, so the run starts from an
#                empty platform and every account gets its predictable
#                email (see api_smoke.py); signs in with QA_USER_PASSWORD.
#   QA_PHOTOS    a folder of NAME.jpg + NAME.thumb.jpg for the showcase;
#                when unset and `sips` is there (macOS), they're made
#                from design/templates.
set -eu
cd "$(dirname "$0")/../.."

API="${QA_API:-http://localhost:8080}"
# The same root account docker-compose.yml creates by default, so the
# local admin login doesn't change after a run.
ADMIN_EMAIL="${QA_ADMIN_EMAIL:-admin@betailo.test}"
ADMIN_PASSWORD="${QA_ADMIN_PASSWORD:-Betailo-dev-admin-2026}"

# Nobody registers into the staff: the API creates this root account at
# startup from its configuration.
export SUPER_ADMIN_EMAIL="$ADMIN_EMAIL" SUPER_ADMIN_PASSWORD="$ADMIN_PASSWORD"
# The limits the smoke test checks against, whatever the shell has set.
export MAX_FARMS_PER_ACCOUNT=1 MAX_FARM_CONTACTS=3 FARM_AUTO_APPROVE=false
# The smoke test signs staff in with a password alone; the second factor
# has its own checks, on an account that sets one up.
export STAFF_MFA_REQUIRED=false

if [ "${QA_RESET:-}" = "1" ]; then
  docker compose up -d mongo
  echo "waiting for MongoDB…"
  for _ in $(seq 1 30); do
    docker compose exec -T mongo mongosh --quiet --eval 'db.runCommand({ping:1}).ok' >/dev/null 2>&1 && break
    sleep 2
  done
  echo "dropping the local database…"
  docker compose exec -T mongo mongosh --quiet betailo --eval 'db.dropDatabase()'
  # The API recreates the collections and the root account at startup.
  docker compose stop backend >/dev/null 2>&1 || true
fi

docker compose up -d --build mongo seaweedfs backend

echo "waiting for the API…"
for _ in $(seq 1 60); do
  curl -sf "$API/health" >/dev/null && break
  sleep 2
done
curl -sf "$API/health" >/dev/null || { docker compose logs backend; exit 1; }

# Showcase photos: re-encoded like the app does, shrinking until each
# fits the upload limits (800 KB for the photo, 120 KB for its copy).
fit() { # src out maxDimension maxBytes
  for dim in "$3" $(($3 * 4 / 5)) $(($3 * 3 / 5)) $(($3 / 2)); do
    for q in 80 70 60; do
      sips -s format jpeg -s formatOptions "$q" -Z "$dim" "$1" --out "$2" >/dev/null
      [ "$(wc -c <"$2")" -le "$4" ] && return 0
    done
  done
}
if [ -z "${QA_PHOTOS:-}" ] && command -v sips >/dev/null 2>&1 && [ -d design/templates ]; then
  QA_PHOTOS="${TMPDIR:-/tmp}/betailo-qa-photos"
  mkdir -p "$QA_PHOTOS"
  for src in design/templates/*; do
    name=$(basename "$src" | sed 's/\.[^.]*$//; s/ /_/g')
    [ -f "$QA_PHOTOS/$name.thumb.jpg" ] && [ "$(wc -c <"$QA_PHOTOS/$name.thumb.jpg")" -le 122880 ] && continue
    fit "$src" "$QA_PHOTOS/$name.jpg" 1600 819200
    fit "$src" "$QA_PHOTOS/$name.thumb.jpg" 640 122880
  done
fi

QA_API="$API" QA_ADMIN_EMAIL="$ADMIN_EMAIL" QA_ADMIN_PASSWORD="$ADMIN_PASSWORD" QA_PHOTOS="${QA_PHOTOS:-}" \
  python3 scripts/qa/api_smoke.py
