# Auth Micro-service

> NATS JetStream + PostgreSQL authentication micro-service (Go 1.25)

## Features

* Register / Login / Logout
* JWT access-token (15 min) + opaque refresh-token (24 h / 30 d)
* Device sessions list & revoke others
* **Automated DB Migrations**: Embedded SQL files applied on startup
* **Dockerized Stack**: Single-command infrastructure setup
* Timing-attack safe password check
* Concurrent-safe session limit (max 4)
* JetStream durability & manual ACK

## Quick Start (Dockerized)

```bash
# 1. Clone
git clone https://github.com/<user>/auth-service.git
cd auth-service

# 2. Env
cp .env.example .env
# edit JWT_ACCESS_SECRET (DATABASE_URL/NATS_URL are pre-configured for Docker)

# 3. Run Stack
# This builds the app, starts Postgres/NATS, and runs migrations automatically
docker-compose up --build

```

Service log:

`INFO auth service started`

## Testing with NATS CLI

```bash
# Register
nats req auth.register '{
  "email":"test@mail.com",
  "password":"12345678",
  "username":"tester"
}'

# Login (save refresh_token)
nats req auth.login '{
  "email":"test@mail.com",
  "password":"12345678",
  "device_name":"laptop",
  "device_type":"desktop",
  "remember_me":false,
  "ip_address":"127.0.0.1",
  "user_agent":"nats-cli"
}'

# Verify access-token
nats req auth.validate '{
  "access_token":"eyJhbGc..."
}'

# Refresh
nats req auth.refresh '{
  "refresh_token":"7fJ9aB..."
}'

# Logout
nats req auth.logout '{
  "session_id":"<uuid>"
}'

```

## Project Layout

```
cmd/auth              → entry point + NATS handlers
internal/config       → fail-safe env loader
internal/data         → models & SQL (Postgres 15+ / UUID)
internal/validator    → input rules
migrations/           → SQL scripts (embedded via go:embed)

```

## Env Variables

| Variable | Description |
| --- | --- |
| `DATABASE_URL` | `postgres://user:password@db:5432/auth_db?sslmode=disable` |
| `NATS_URL` | `nats://nats:4222` |
| `JWT_ACCESS_SECRET` | **32+ chars** for signing tokens |

## API Contract

Full message schema below.

---

### 1. auth.register

**Goal**: create a new user.

**Request (client → NATS)**

```json
{
  "email": "string",      // valid email address
  "password": "string",   // 8-72 bytes
  "username": "string"    // 4-100 characters
}

```

**Success 201**

```json
{
  "status": 201,
  "data": "user successfully created"
}

```

**Validation errors 422**

```json
{"status":422,"error":{"username":"must be provided"}}
{"status":422,"error":{"email":"must be a valid address"}}
{"status":422,"error":{"password":"must be at least 8 bytes long"}}

```

**Business error 409**

```json
{"status":409,"error":{"email":"is already in use"}}

```

---

### 2. auth.login

**Goal**: obtain tokens and list active sessions.

**Request**

```json
{
  "email": "string",
  "password": "string",
  "device_name": "string",
  "device_type": "string",   // desktop|mobile|tablet
  "remember_me": false,      // false = 24 h, true = 30 d
  "ip_address": "string",
  "user_agent": "string"
}

```

**Success 200**

```json
{
  "status": 200,
  "data": {
    "refresh_token": "7fJ9aB...",
    "access_token": "eyJhbGc...",
    "current_session": {
      "session_id": "uuid",
      "device_name": "string",
      "last_used_at": "2025-11-30T18:34:37Z"
    },
    "other_sessions": []
  }
}

```

---

### 3. auth.validate

**Goal**: validate access-token and return claims.

**Success 200**

```json
{
  "status": 200,
  "data": {
    "id": "uuid",
    "email": "string",
    "username": "string"
  }
}

```

---

### 4. auth.refresh

**Goal**: obtain a new access-token using the opaque refresh-token.

**Success 200**

```json
{
  "status": 200,
  "data": {
    "access_token": "eyJnew..."
  }
}

```

---

### 5. auth.logout

**Goal**: revoke a single device session.

**Success 200**

```json
{
  "status": 200,
  "data": "user successfully logged out"
}

```

---

### Common Rules

* All subjects are part of **JetStream** stream `auth` (WorkQueue policy).
* Responses are **always** published to `msg.Respond` (inbox).
* Timestamps are RFC-3339 UTC.
* Access-token TTL: **15 min**; refresh-token: **24 h** (or 30 d if `remember_me=true`).
* **Maximum 4 active sessions**; older ones are auto-revoked.

---