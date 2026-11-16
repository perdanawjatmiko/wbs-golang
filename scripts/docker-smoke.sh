#!/bin/sh
# Read-only functional checks. Explicitly selected development DB must already be migrated/seeded.
set -eu
if [ "${WBS_APPROVED_DEV_DATABASE:-}" != yes ]; then
  echo 'BLOCKED: set WBS_APPROVED_DEV_DATABASE=yes only for an identified, approved development database.' >&2
  exit 2
fi
if [ ! -f .env.docker ]; then echo 'BLOCKED: private .env.docker required.' >&2; exit 2; fi
compose() { docker compose --env-file .env.docker "$@"; }
compose config --quiet
compose exec -T backend /app/check live
compose exec -T backend /app/check ready
compose exec -T backend /app/check db
compose exec -T worker /app/check db
compose exec -T backend /app/migrate status
compose exec -T frontend node -e "fetch('http://127.0.0.1:3000/health/live',{signal:AbortSignal.timeout(2000)}).then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))"
echo 'PASS: read-only health/connectivity/status smoke. Auth/workflow/file/SSE checks still require explicit isolated QA fixtures.'
