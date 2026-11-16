# Local Docker development

This is a local environment, not production deployment or validation. Exactly three services: frontend, backend, worker. PostgreSQL remains an existing managed host/private-VM service. No PostgreSQL/Redis image, service, port mapping, data volume, initialization script or proxy is included.

## Images and runtime

backend/Dockerfile builds api, worker, migrate, seed and check with Go1.26.11 (security patch within existing1.26 toolchain line). API and worker share wbs-backend:local. The scratch runtime has no shell/compiler/package manager, runs10001:10001, and includes CA roots/timezone data. Frontend uses Node22.23.5 (official Node22 security release2026-11-11), npm ci with lock integrity/lifecycle scripts disabled, Next standalone and only traced runtime files/static/public output. Dependency installation requires registry access during a normal image build; audit validation must instead use prepared offline dependencies in a separately constrained sandbox. Google Geist font downloading is an existing build prerequisite, not removed in this phase. Base tags are documented patch versions, not latest; registry digests should be recorded/pinned after the images can actually be built/scanned.

Both contexts exclude private env files, generated output and local attachments. No secret build arguments are used. Runtime Compose drops all capabilities, enables no-new-privileges, uses non-root/read-only filesystems, and bounds memory/CPU/pids/tmpfs. /tmp is the dedicated writable temporary directory. The only persistent volume is attachments, mounted only by API at /var/private/attachments. Image directory ownership10001 and mode0700 support volume copy-up; file creation stays0600. Actual volume initialization/persistence still requires the blocked runtime smoke test. Worker needs no attachment mount and has no HTTP port.

## Configuration

```sh
cp .env.docker.example .env.docker
# Edit privately: identify an approved existing DEVELOPMENT database; replace all placeholders.
# No database connection/migration/seed is made by copying this file.
docker compose --env-file .env.docker config --quiet
docker compose --env-file .env.docker build
docker compose --env-file .env.docker up -d
```

Use the existing DATABASE_URL, not incompatible DB_HOST/DB_PASSWORD variables. Percent-encode credentials. Inside a container127.0.0.1 is that container. For macOS-host PostgreSQL use host.docker.internal; for a VM use its private IP/private DNS. Linux host-gateway mapping is explicit; no host networking. Frontend server uses http://backend:8080 internally; browser uses APP_ORIGIN, default http://localhost:3000. If changing FRONTEND_PORT, change APP_ORIGIN and CORS_ALLOWED_ORIGINS together. Bindings remain127.0.0.1:3000/8080 by default. Never put DATABASE_URL/provider credentials in NEXT_PUBLIC values.

Compose forces SMTP_ENABLED=false and TELEGRAM_ENABLED=false for local QA, irrespective of env-file values. Pending delivery rows are not falsely marked sent. Backend/worker read private .env.docker; frontend receives only selected nonsecret settings. Do not place bootstrap credentials in the normal service env file. WBS_ENV_FILE can select another private backend env-file path; it is a Compose control, not a Go database setting.

## External PostgreSQL and approval boundary

Treat the server/database as managed resources. Nothing in startup migrates, seeds, creates/resets a database or changes PostgreSQL/firewall configuration. The following commands are explicit operator actions. Run only after the owner identifies/approves the development database:

```sh
# Read-only connectivity checks; --no-deps avoids starting other services.
docker compose --env-file .env.docker run --rm --no-deps backend /app/check db
docker compose --env-file .env.docker run --rm --no-deps worker /app/check db
# Read-only migration metadata, no DDL:
docker compose --env-file .env.docker run --rm --no-deps backend /app/migrate status
# Explicit approved development database only:
docker compose --env-file .env.docker run --rm --no-deps backend /app/migrate up
docker compose --env-file .env.docker run --rm --no-deps backend /app/seed
```

Initial admin bootstrap uses existing ADMIN_SEED_EMAIL/PASSWORD/NAME. Export them privately in the current shell and pass variable names, never literal passwords in commands:

```sh
docker compose --env-file .env.docker run --rm --no-deps \
  -e ADMIN_SEED_EMAIL -e ADMIN_SEED_PASSWORD -e ADMIN_SEED_NAME backend /app/seed
```

No public default password and no existing password overwrite. No down/reset command is part of normal local setup.

## Network, authentication and TLS troubleshooting

Connection refused: check the chosen host is not container127.0.0.1, server is already running, private route/VPN exists, and PostgreSQL listens on its intended private interface. A timeout often indicates routing/firewall rather than authentication. Worker startup retries5 times with1s exponential delay capped30s, respects DB_CONNECT_TIMEOUT and signal cancellation, then exits visibly; runtime polling tolerates temporary DB outages. Compose restart remains a local resilience policy, not unlimited transaction retries.

Have the DBA inspect listen_addresses, firewall and pg_hba.conf without changing them automatically. Identify the source IP PostgreSQL actually sees using owner-observed connection logs or an approved session query `SELECT inet_client_addr(), inet_server_addr();`; Docker NAT may present the host/VM gateway rather than a container IP. Only then recommend a narrow rule, for example `hostssl APPROVED_DEV_DB wbs_app OBSERVED_SOURCE_IP/32 scram-sha-256` (IPv6 uses /128). Never use0.0.0.0/0 or an assumed whole Docker subnet as an access rule. The DBA must separately approve and apply any exact rule/interface/firewall adjustment; this phase did not execute one.

Authentication/pg_hba errors: verify approved database/role, URL encoding, actual observed source, SCRAM policy and TLS mode. Application runtime role should have CONNECT, schema USAGE and only required table/sequence DML; use a separately approved migration role for DDL, never a superuser for API/worker. Actual privileges were not inspected.

Use sslmode=verify-full for a private VM when TLS is available; hostname must match certificate SAN. System CA roots are included. For a private CA, arrange an explicit read-only Compose config at a known container path, then reference that path with sslrootcert in DATABASE_URL. Do not mount a host home directory or disable certificate verification to hide a CA/hostname problem. APP_ENV=production requires verify-full, but production operation is out of scope. Local host-only sslmode=disable is an explicit development exception, not a claim that private networks are trustworthy.

## Health, lifecycle and persistence verification

API check live is process liveness; check ready performs bounded database readiness. Frontend /health/live is process-only JSON. Worker has no fake HTTP health server. depends_on orders worker/frontend after API health; it does not prove continued DB availability. SIGTERM drains API/SSE, cancels worker work, and closes pools. stop_grace_period30s exceeds default HTTP15s shutdown. /tmp is ephemeral; attachments volume persists across normal stop/restart/down without volume deletion.

```sh
python3 scripts/check-docker-config.py  # static policy only; no services or DB calls
# Once owner approval + identified migrated/seeded dev DB are explicit:
WBS_APPROVED_DEV_DATABASE=yes sh scripts/docker-smoke.sh
```

The read-only smoke checks health, connectivity and migration metadata. Separate explicitly authorized isolated QA HTTP fixtures are needed for login/CRUD/report/status/attachment/SSE/outbox behavior. They create QA records; never point them at unidentified data. Restart persistence should upload one dummy fixture through authorized API, verify download, restart API, then verify the same bytes. Do not delete user volumes or real reports. No such runtime/persistence check was executed here.

## Current verification blocker

Docker daemon/Compose are available. Base-image pull and backend build failed with `no space left on device` inside Docker Desktop storage. Frontend base-image extraction failed too. Existing images, build cache, containers and volumes were not deleted or pruned. No application container was started and no external database target was accessed. Owner-managed disk remediation is required before build/smoke verification can resume. Inspect Docker Desktop storage settings and owned disposable resources; do not blindly prune volumes. No production, load/performance or real provider testing is included.
