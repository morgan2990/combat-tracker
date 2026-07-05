## Why

A production outage investigation (SSH into the Raspberry Pi host, direct inspection of `docker ps`, container logs, `docker events`, and running processes) found that only 2 of the 6 processes the deployed app depends on are supervised: `combatapp` and `portainer` run under Docker Compose with a restart policy, but `mongod`, `minio`, `typesense-server`, and `cloudflared` are all bare host processes started by hand, with no systemd unit, no Docker, and no restart policy of any kind. If any of the four unsupervised processes dies for any reason (OOM, a dropped SSH session that started it without `nohup`/`disown`, a power blip, a crash), nothing brings it back — the Pi requires manual SSH access and a manual restart to recover. No panic or crash was found anywhere in the Go application code itself (grepped for `recover()`, inspected `combatapp`'s logs directly), so the app's crash-worthiness is not the source of the reported instability.

This gap produces exactly the symptom reported ("the app seems to be crashing or becoming not accessible"), through two distinct failure modes depending on which unsupervised process goes down: `cloudflared` dying disconnects the tunnel entirely (Cloudflare-side 1033/530 errors, app fully unreachable), while `mongod`/`minio`/`typesense-server` dying leaves `combatapp` running but every database-touching request timing out after its 5s context deadline (app "up" but effectively broken). Neither failure mode self-heals today.

## What Changes

- Bring `mongod`, `minio`, `typesense-server`, and `cloudflared` under the same kind of process supervision `combatapp` and `portainer` already have, so a crash in any of them is followed by an automatic restart with no manual intervention.
- Move this infrastructure into version control. None of the four services' configuration (data paths, ports, tunnel token, API keys) currently exists in any committed file — it lives only as ad hoc command lines someone once typed on the Pi. Consolidating into the project's `docker-compose.yml` (preferred) or committed systemd unit files makes the deployment reproducible and reviewable.
- Add a minimal persistent record of container/process lifecycle events (start/stop/restart, with timestamps and exit reasons) so a future incident can be diagnosed by reading a log instead of the multi-hour manual forensic reconstruction (SSH, `docker events`, `journalctl`, process inspection) this investigation required.
- Explicitly out of scope: full APM/tracing tools (e.g. New Relic) were considered and deprioritized — they instrument the application layer, which this investigation showed is not where the fragility lives.

## Capabilities

### New Capabilities
- `deployment-supervision`: defines the operational requirement that every process the app depends on (app server, database, object storage, search index, tunnel) must be automatically restarted on failure, and that container/process lifecycle events are persistently recorded.

### Modified Capabilities
- None. No existing spec covers deployment/infrastructure behavior — this is a new, purely operational capability with no user-facing requirement changes.

## Impact

- `docker-compose.yml`: add services for `mongod`, `minio`, `typesense-server`, and `cloudflared` (or document why any of them stay outside Compose), each with a restart policy and health check; add `depends_on` so `combatapp` waits on its dependencies being healthy.
- Possibly new systemd unit files, if any service is kept as a native process rather than moved into Compose.
- New: a lifecycle-event logging mechanism (e.g. `docker events` piped to a rotated log file) and, for any service kept outside Docker, an equivalent supervision/logging story.
- No changes expected to Go application source (`main.go`, `api/`, `room/`, `store/`, `ws/`) — no bug was found there.
- Secrets currently only present as environment variables / bare command-line flags on the Pi (Mongo credentials, Typesense API key, Minio keys, cloudflared tunnel token) will need to move into the committed Compose file's env handling (e.g. via the existing `.env` file already referenced by `docker-compose.yml`) without checking secrets themselves into git.
