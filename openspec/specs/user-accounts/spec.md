# Spec: User Accounts

## Purpose

Defines how visitors self-serve create accounts, log in, log out, and how sessions are authenticated across HTTP and WebSocket, including the dashboard data endpoint used by the frontend.

## Requirements

### Requirement: A visitor can self-serve create a User account

The system SHALL provide a `POST /api/signup` endpoint that creates a new `User` from a `username` and `passphrase`. The passphrase SHALL be hashed with bcrypt before storage; the plaintext passphrase SHALL NOT be persisted anywhere. `username` SHALL be unique across all users.

#### Scenario: Successful signup
- **WHEN** a client sends `POST /api/signup` with `{ "username": "aria", "passphrase": "correct-horse" }` and no `User` with that username exists
- **THEN** the server SHALL create a `User` document with a bcrypt hash of the passphrase, create a `Session` for the new user, set the session cookie, and respond with HTTP 201

#### Scenario: Signup rejected for duplicate username
- **WHEN** a client sends `POST /api/signup` with a `username` that already exists
- **THEN** the server SHALL respond with HTTP 409 and SHALL NOT create a new `User`

#### Scenario: Signup rejected for empty username or passphrase
- **WHEN** a client sends `POST /api/signup` with an empty `username` or `passphrase`
- **THEN** the server SHALL respond with HTTP 400 and SHALL NOT create a new `User`

### Requirement: A user can log in with username and passphrase

The system SHALL provide a `POST /api/login` endpoint that verifies a `username`/`passphrase` pair against the stored bcrypt hash.

#### Scenario: Successful login
- **WHEN** a client sends `POST /api/login` with a `username` and the correct `passphrase`
- **THEN** the server SHALL create a `Session`, set the session cookie, and respond with HTTP 200

#### Scenario: Login rejected for wrong passphrase
- **WHEN** a client sends `POST /api/login` with a `username` that exists and an incorrect `passphrase`
- **THEN** the server SHALL respond with HTTP 401 and SHALL NOT create a `Session`

#### Scenario: Login rejected for unknown username
- **WHEN** a client sends `POST /api/login` with a `username` that does not exist
- **THEN** the server SHALL respond with HTTP 401 and SHALL NOT create a `Session`

### Requirement: Sessions are cookie-based, DB-backed, and roll forward on activity

A successful signup or login SHALL set an `HttpOnly`, `SameSite=Lax` cookie carrying an opaque session token. The `Secure` attribute SHALL be controlled by a server-side configuration flag (defaulting to enabled) so the cookie can be used over plain HTTP in local/LAN development. Each `Session` document SHALL store `token`, `user_id`, `created_at`, `last_seen_at`, and `expires_at`, with `expires_at` set to `last_seen_at` plus a rolling window of approximately 90 days.

#### Scenario: Session expiry rolls forward on use
- **WHEN** an authenticated request arrives bearing a valid, non-expired session cookie, and the session's `last_seen_at` is more than 5 minutes in the past
- **THEN** the server SHALL update `last_seen_at` and recompute `expires_at` as `last_seen_at + 90 days`

#### Scenario: Recently-touched session is not re-written
- **WHEN** an authenticated request arrives and the session's `last_seen_at` is less than 5 minutes in the past
- **THEN** the server SHALL NOT perform a write to update the session, to avoid excessive database writes during active use

#### Scenario: Expired session is treated as logged out
- **WHEN** a request arrives with a session cookie whose `expires_at` is in the past
- **THEN** the server SHALL treat the request as unauthenticated, regardless of whether the `Session` document still exists

#### Scenario: WebSocket connections authenticate via the same cookie
- **WHEN** a browser opens a WebSocket upgrade request to `/ws`
- **THEN** the session cookie SHALL be sent automatically as part of the upgrade request (same-origin), and the server SHALL resolve the connecting user's identity from it exactly as it would for an HTTP request

### Requirement: A user can log out

The system SHALL provide a `POST /api/logout` endpoint that invalidates the current session.

#### Scenario: Logout deletes the session
- **WHEN** an authenticated client sends `POST /api/logout`
- **THEN** the server SHALL delete the corresponding `Session` document and respond with a cleared session cookie

### Requirement: A logged-in user can fetch their own dashboard data

The system SHALL provide a `GET /api/me` endpoint that returns the authenticated user's identity along with the rooms they own and the PCs they own, for use by the frontend dashboard.

#### Scenario: Authenticated request returns user data
- **WHEN** an authenticated client sends `GET /api/me`
- **THEN** the server SHALL respond with HTTP 200 and a JSON body containing the user's `username`, the rooms where they are `owner_user_id`, and the PCs where they are `owner_user_id`

#### Scenario: Unauthenticated request is rejected
- **WHEN** a client sends `GET /api/me` without a valid session cookie
- **THEN** the server SHALL respond with HTTP 401
