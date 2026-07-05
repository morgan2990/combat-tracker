## ADDED Requirements

### Requirement: Automatic restart of dependent services on failure
Every process the deployed application depends on to serve requests (database, object storage, search index, and public tunnel) SHALL be supervised such that an unexpected exit is followed by an automatic restart, with no manual/SSH intervention required.

#### Scenario: A dependency process exits unexpectedly
- **WHEN** any of the database, object storage, search index, or tunnel processes terminates for any reason (crash, signal, OOM)
- **THEN** the process is automatically restarted without any human action, within the same order of magnitude of time as the application server's own existing restart behavior

#### Scenario: Host reboots
- **WHEN** the Raspberry Pi host restarts (power cycle, planned reboot)
- **THEN** every dependent service comes back up automatically alongside the application server, with no manual step required to restore full functionality

### Requirement: Application waits for dependencies to be ready
The application server SHALL NOT begin accepting traffic in a state where its database, object storage, or search index dependencies are unreachable in a way that causes every request to fail.

#### Scenario: Dependencies are still starting up
- **WHEN** the application server's process starts at the same time as its database/storage/search dependencies (e.g. after a host reboot)
- **THEN** the application server does not begin serving traffic until those dependencies report healthy, rather than accepting connections and failing every dependent request

### Requirement: Deployment configuration is version-controlled
The complete configuration needed to stand up every service the application depends on (ports, data paths, restart policy, health checks) SHALL exist in files committed to the project's repository, with secrets excluded from version control but referenced from a documented location.

#### Scenario: Reproducing the deployment on a new host
- **WHEN** someone needs to stand up the full stack on a new or replacement host
- **THEN** they can do so from the committed configuration files plus a documented set of secret values, without needing to reconstruct any service's configuration by inspecting a running process

#### Scenario: Secrets are not committed
- **WHEN** the deployment configuration is committed to the repository
- **THEN** no credential, API key, or tunnel token appears in the committed files themselves

### Requirement: Persistent record of service lifecycle events
Start, stop, and restart events for every supervised service SHALL be recorded to a log that persists independently of any single service's or the container runtime's own in-memory state, with a defined retention window.

#### Scenario: Diagnosing an incident after the fact
- **WHEN** a service has restarted one or more times and someone investigates after the fact
- **THEN** they can determine which service(s) restarted and when from a persisted log, without needing to reconstruct the timeline from live process inspection

#### Scenario: The container runtime itself restarts
- **WHEN** the underlying container runtime restarts (which clears its own in-memory event history)
- **THEN** lifecycle events recorded before that restart remain available in the persistent log
