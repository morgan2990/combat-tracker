## Context

The production deployment runs on a single Raspberry Pi 5. Today, `combatapp` and `portainer` are the only services managed by Docker Compose (`docker-compose.yml`, `restart: unless-stopped`). `mongod`, `minio`, `typesense-server`, and `cloudflared` were each started by hand at some point (likely across different SSH sessions — their PIDs and start times don't correlate) and have run unsupervised since. This was discovered through direct SSH investigation, not from any existing documentation, because none of this configuration exists in a committed file anywhere.

An earlier attempt to containerize `cloudflared` specifically was abandoned: three short-lived `cloudflare/cloudflared` containers exist on the host, each exiting within minutes because the tunnel config pointed at `http://localhost:8080`, which inside an isolated container network doesn't resolve to the `combatapp` container. Whoever tried this reverted to running `cloudflared` as a bare process instead of fixing the networking. Any redesign has to solve this specific problem or it will fail the same way again.

`mongod`, `minio`, and `typesense-server` all have existing data directories on the Pi's filesystem. Whatever supervision mechanism is chosen must not lose or orphan that data during cutover.

Four standalone compose files (`mongodb-compose.yml`, `minio-compose.yml`, `typesense-compose.yml`, `cloudflared-compose.yml`) exist in the repo root, one per service, matching an earlier attempt at this same hardening work. None of their services run today: every port they declare (`mongod`→27017, `minio`→9000/9001, `typesense-server`→8108) is already bound by the corresponding native process, so each stack fails to bind its ports — a straightforward migration-ordering problem the tasks below account for, not a mystery.

Their bind mounts, checked against the live data on disk:
- `mongodb-compose.yml` (`/mnt/data/mongodb:/data/db`) and `minio-compose.yml` (`/mnt/data/minio:/data`) point at exactly where the live, actively-written data already is — low-risk to adopt as-is.
- `typesense-compose.yml` uses a fresh relative path (`./typesense-data:/data`), which does **not** match the native process's actual `--data-dir /data`. `/data` on the host doesn't look like a normal Typesense data directory (no index files, just a stray `compose/` folder) — what's actually in the live index needs confirming (or a rebuild accepted) before this is settled; see Open Questions.
- All four files hardcode credentials in plaintext (Mongo admin/password, Minio access/secret keys, the Typesense API key, and the full `cloudflared` tunnel token). They're excluded via `.gitignore`, and Decision 5 (secrets in `.env`) applies to them the same as anything else.

Decision 1 folds their content into `docker-compose.yml` rather than writing the services from scratch.

## Goals / Non-Goals

**Goals:**
- Every process `combatapp` depends on restarts automatically on failure, with no SSH/manual step.
- The full deployment (all services, ports, restart policies, health checks) is defined in one or more files committed to this repo.
- `combatapp` doesn't accept traffic (or at least doesn't fail every request) before its dependencies are actually ready.
- A future incident can be diagnosed by reading a log file, not by reconstructing events via `docker events`, `journalctl`, and process inspection over SSH.
- Existing Mongo/Minio/Typesense data survives the migration into supervision unchanged.

**Non-Goals:**
- Full APM/tracing instrumentation of the Go application (New Relic, Sentry, etc.) — explicitly deferred; not where the investigated fragility lives.
- Multi-node/HA deployment — this stays a single-Pi deployment.
- Resource-usage dashboards or long-term metrics retention (Prometheus/Grafana) — lifecycle-event logging here is intentionally minimal, not a metrics platform.
- Changing the application's actual data-access code/timeouts in `store/*.go` — out of scope, no bug was found there.

## Decisions

### 1. Merge all five services into the one existing `docker-compose.yml`
`mongo`, `minio`, `typesense`, and `cloudflared` are folded into the existing `docker-compose.yml` as services alongside `combatapp`, using the images specified in the standalone compose files (`mongo:latest`, `minio/minio`, `typesense/typesense:29.1-arm64-lg-page16`, `cloudflare/cloudflared:latest`) — one file, one Compose project, instead of either systemd units or four independent Portainer stacks.

**Why:** `combatapp` and `portainer` already prove the Compose + `restart: unless-stopped` pattern works reliably on this host. A single project gives one place to see the whole stack, one restart-policy mechanism, and — critically — Compose's `depends_on: condition: service_healthy` and automatic service-name DNS resolution **only work within one project**. Keeping the four services as separate Portainer deployments would mean giving up native health-gated startup ordering and needing an explicit shared external Docker network just so `cloudflared` can reach `combatapp` by name — solvable, but strictly more moving parts for no real benefit given this is a single-Pi deployment with one operator.

**Alternatives considered:**
- *systemd units with `Restart=always` for each.* Rejected: more files to maintain, no native health-gated dependency ordering, doesn't share infrastructure with the pattern already proven on this box.
- *Keep the four services as independent Compose projects (matching their current standalone compose files).* Rejected in favor of merging: independent stacks can still be redeployed/updated individually if desired later (nothing about merging prevents running `docker compose up -d <service>` for just one), but merging is what actually fixes the `cloudflared`-can't-reach-`combatapp` networking bug and gets real dependency-ordering, both of which independent stacks would need extra machinery to achieve.

### 2. Fix `cloudflared`'s networking by pointing it at the Compose service name, not `localhost`
The tunnel's ingress config changes from `http://localhost:8080` to `http://combatapp:8080`, relying on Compose's default bridge network's internal DNS (service name resolution). `cloudflared` joins the same Compose network as `combatapp`.

**Why:** This is the exact, identified cause of the previous containerization attempt's failure. Using the service name instead of `localhost` (or alternatively `--network host`) resolves it without needing host networking mode, which is preferable on a Pi where other host-network services (SSH, Portainer's own ports) shouldn't be put at risk of port collisions.

**Alternative considered:** `network_mode: host` for the `cloudflared` service, avoiding the DNS question entirely. Rejected as the default choice because it's a bigger blast-radius change (shares the host's entire network namespace) for a problem that Compose's built-in service discovery already solves cleanly; kept as a fallback if service-name resolution has any issue in practice. Because all services live in one merged Compose project (Decision 1), service-name resolution is the default — no cross-project external network needed.

### 3. Bind-mount existing data directories into the new containers
`mongo`'s volume maps to `/mnt/data/mongodb:/data/db` and `minio`'s to `/mnt/data/minio:/data` — exactly where the current native processes' active data already lives. `typesense`'s mount needs resolving before cutover: `typesense-compose.yml` points at a fresh relative path, not the native process's actual `--data-dir /data` (see Open Questions). The migration stops each native process, starts the equivalent container against the same on-disk data, and verifies before removing the old binary/process.

**Why:** Avoids a data migration/export-import step and its associated risk for Mongo and Minio. This is a lift-and-shift of the process boundary, not the data. Typesense may end up being a rebuild-from-source rather than a lift-and-shift, given the app already has a monster-scrubber capability that can repopulate the search index — acceptable if the existing index turns out to have nothing unrecoverable in it.

### 3a. Stop each native process before starting its container replacement
Every one of `mongod`, `minio`, and `typesense-server`'s ports (27017, 9000/9001, 8108 respectively) is already bound by the native process, which is why those stacks' containers cannot start while the native processes are running. Each migration step must stop the native process first and confirm the port is free before bringing up the container; there is no way to run both simultaneously to compare them side by side.

**Why:** Docker cannot bind a host port that's already in use by another process; this isn't a Docker/Compose configuration problem to solve, it's a required ordering step.

### 4. Lifecycle-event logging via `docker events`, not a metrics stack
A small long-running process (a `systemd` unit, or a minimal container) runs `docker events --format '{{json .}}'` and appends to a rotated log file (`logrotate` config, e.g. 14-day retention).

**Why:** `docker events`' own in-memory buffer is wiped on daemon restart — the exact gap that made this incident's investigation take hours instead of minutes. Piping it to disk costs almost nothing and directly targets that gap. A full metrics stack (Prometheus/cAdvisor/Grafana) would add real value (resource trends) but is heavier than what's needed to close the specific gap this incident exposed; can be revisited later if resource-pressure questions come up again.

### 5. Secrets stay in the existing `.env` file, not in `docker-compose.yml`
Mongo root credentials, the Minio access/secret keys, the Typesense API key, and the `cloudflared` tunnel token move from ad hoc command-line flags into `.env` (already `.gitignore`d, already referenced by `docker-compose.yml`'s `env_file: .env`), referenced from the compose file via `${VARIABLE}` interpolation.

**Why:** Keeps the existing secret-handling convention (`.env` is already excluded from git and already used this way for `combatapp`) rather than introducing a second mechanism.

### 6. Watchdog timer as a safety net over Docker's own restart policy
A systemd user timer (`docker-watchdog.timer`, every 30s) runs `docker compose up -d` in the project directory, logging each run. This is in addition to, not instead of, each container's own `restart: unless-stopped` policy.

**Why:** Testing during implementation (task 5.3) found that `restart: unless-stopped` does not reliably fire on this Pi's rootless Docker setup (rootlesskit + gVisor-tap-vsock networking) — `docker kill` against any of `mongo`, `minio`, `typesense`, or `cloudflared` left the container down indefinitely (`RestartCount` stayed at 0 across repeated, isolated tests, no daemon restart attempt observed even after 90+ seconds). The exact root cause wasn't pinned down (daemon log capture over SSH was unreliable on this setup), but since `docker kill` sends the same SIGKILL an OOM-kill or crash would, this is a real gap for the actual failure modes this change exists to fix. `docker compose up -d` is idempotent — a no-op for already-running, unchanged services, and a start for anything stopped or exited — so running it on a timer is a safe, low-effort way to get the actual required behavior (auto-recovery) regardless of whether the underlying restart-policy mechanism can be trusted on this host. Verified working: killed `typesense`, watchdog brought it back within ~25 seconds with no manual step.

## Risks / Trade-offs

- **[Risk]** Cutting over a stateful service (Mongo especially) has a downtime window and a small chance the container fails to start against the bind-mounted data (permissions/UID mismatch between the native process's user and the container's internal user). → **Mitigation**: migrate one service at a time, off of active-usage hours, verify `combatapp` can reach the new container before decommissioning the native process; keep the native binary in place (just stopped, not removed) until the container has run successfully for a period.
- **[Risk]** Changing `cloudflared`'s ingress target from `localhost:8080` to `combatapp:8080` reintroduces the exact failure mode from the earlier abandoned attempt if the Compose network/service-name resolution doesn't behave as expected. → **Mitigation**: validate connectivity (`docker compose exec cloudflared wget -qO- http://combatapp:8080`) before cutting the tunnel over; keep `network_mode: host` as a documented fallback.
- **[Risk]** Health checks that are too strict (e.g. Mongo not accepting connections for a few seconds after start) could make `combatapp` wait longer than necessary or fail its own health check during Pi boot. → **Mitigation**: tune `start_period`/`retries` on each healthcheck generously given the Pi's modest hardware.
- **[Trade-off]** `docker events`-to-file is not a full observability solution — it won't explain *why* a container died if the cause is external to Docker (e.g. an OOM at the OS level would still need `dmesg`/`journalctl`, as in this investigation). Accepted for now given the Non-Goals; revisit if another incident isn't fully explained by this log alone.

## Migration Plan

1. Merge `mongo`, `minio`, `typesense`, `cloudflared` into the existing `docker-compose.yml` as services, using the standalone compose files as the starting point for each service's config, with bind mounts pointing at the current data locations, health checks, and `combatapp`'s `depends_on` updated to require them healthy.
2. Move all secrets/config (Mongo credentials, Minio keys, Typesense API key, `cloudflared` tunnel token) out of the standalone compose files and into `.env`.
3. One service at a time, in order of increasing risk — `cloudflared` (stateless) → `minio` → `typesense` (data-path question to resolve first) → `mongo` (most critical, most data) last:
   a. Stop the native process and confirm its port is actually free (`ss -tln`).
   b. Bring up the new container against the existing data path.
   c. Verify the app / tunnel works end-to-end.
   d. Only then move to the next service.
4. Set up the `docker events`-to-file logger and logrotate config.
5. Remove the three dead `cloudflare/cloudflared` container husks left over from the earlier abandoned attempt.
6. Document the full stack (this proposal + design doc serve as that documentation going forward).

**Rollback:** at each step, the native process/binary is left installed (just stopped) until its container replacement has proven stable, so reverting a single service is "stop the container, restart the native process" without needing to touch anything else already migrated.

## Open Questions

- ~~What's actually in the live Typesense index at `/data`?~~ Resolved: the index holds real, non-rebuildable data (14,365 monsters including custom/private user-owned ones, 2,640 items). `/data` is an OS-level bind mount to `/data/compose/2/typesense-data`, which is the real path the merged compose file's `typesense` service must bind-mount.
- Is 14 days of retention enough for the `docker events` log, or should it be longer given how infrequently incidents happen? Default to 14 days, adjustable trivially since it's just a `logrotate` setting.
- Are there other native processes on this Pi outside this investigation's scope (e.g. anything else Portainer manages) that should get the same treatment? Not discovered during this investigation; worth a quick audit but not blocking this change.
