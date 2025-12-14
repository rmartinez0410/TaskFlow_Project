
```
# Auth Micro-service
> NATS JetStream + PostgreSQL authentication micro-service (Go 1.25)

## Features
- Register / Login / Logout
- JWT access-token (15 min) + opaque refresh-token (24 h / 30 d)
- Device sessions list & revoke others
- Timing-attack safe password check
- Concurrent-safe session limit (max 4)
- JetStream durability & manual ACK

## Quick Start (local)
```bash
# 1. Clone
git clone https://github.com/etternalattack/auth-service.git
cd auth-service

# 2. Env
cp .env.example .env
# edit AUTH_DB_DSN and JWT_ACCESS_SECRET


# 4. Migrate
create db auth (postgres)
migrate -path=./migrations/ -database=$AUTH_DB_DSN up

# 5. Run
go run ./cmd/auth
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

##  Project Layout
```
cmd/auth              → entry point + handlers
internal/data         → models & SQL
internal/validator    → input rules
migrations/           → SQL up/down
```

##  Env Variables
| Variable         | Example value |
|------------------|---------------|
| `AUTH_DB_DSN`    | `postgres://auth:password@localhost:5432/auth?sslmode=disable` |
| `JWT_ACCESS_SECRET` | **32+ chars** (`pei3einoh0Beem6uM6Ungohn2heiv5lah1ael4joopie5JaigeikoozaoTew2Eh6`) |

---

## API_CONTRACT

```
# NATS Auth Service – Message Contract

Subjects listened: `auth.*` (JetStream stream `auth`, WorkQueue policy)
All responses are JSON and **replied** through `msg.Respond`.

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
{"status":422,"error":{"username":"must not be less than 4 characters"}}
{"status":422,"error":{"username":"must not be more than 100 characters"}}
{"status":422,"error":{"email":"must be provided"}}
{"status":422,"error":{"email":"must be a valid address"}}
{"status":422,"error":{"password":"must be provided"}}
{"status":422,"error":{"password":"must be at least 8 bytes long"}}
{"status":422,"error":{"password":"must not be more than 72 bytes long"}}
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
  "password": "string",      // 8-72 bytes
  "device_name": "string",
  "device_type": "string",   // desktop|mobile|tablet|...
  "remember_me": false,      // false = 24 h, true = 30 d
  "ip_address": "string",    // client IP
  "user_agent": "string"     // client UA
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
      "device_type": "string",
      "last_used_at": "2025-11-30T18:34:37Z"
    },
    "other_sessions": [
      {
        "session_id": "uuid",
        "device_name": "string",
        "device_type": "string",
        "last_used_at": "2025-11-30T18:30:00Z"
      }
    ]
  }
}
```

**Errors**
| status | data |
|--------|------------|
| 400 | `{ "email": "must be a valid address", "password": "must be at least 8 bytes long" }` |
| 401 | `"invalid credentials"` (generic, no user/pass distinction) |

---

### 3. auth.validate
**Goal**: validate access-token and return claims.

**Request**
```json
{
  "access_token": "eyJhbGc..."
}
```

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

**Errors**
| status | data |
|--------|------------|
| 401 | `true` (malformed / wrong signature) |
| 401 | `false` (expired) |
| 500 | `"internal server error"` (rare) |

---

### 4. auth.refresh
**Goal**: obtain a new access-token using the opaque refresh-token.

**Request**
```json
{
  "refresh_token": "7fJ9aB..."
}
```

**Success 200**
```json
{
  "status": 200,
  "data": {
    "access_token": "eyJnew..."
  }
}
```

**Errors**
| status | data |
|--------|------------|
| 401 | `"invalid token"` (not found or revoked) |
| 401 | `"token expired"` |
| 500 | `"internal server error"` |

---

### 5. auth.logout
**Goal**: revoke a single device session.

**Request**
```json
{
  "session_id": "uuid"
}
```

**Success 200**
```json
{
  "status": 200,
  "data": "user successfully logged out"
}
```

**Errors**
| status | data |
|--------|------------|
| 404 | `"session not found"` (already revoked or non-existent) |
| 500 | `"internal server error"` |

---

### Common Rules
- All subjects are part of **JetStream** stream `auth` (WorkQueue policy).
- Responses are **always** published to `msg.Respond` (inbox) and **manually ACKed**.
- Timestamps are RFC-3339 UTC.
- Access-token TTL: **15 min**; refresh-token: **24 h** (or 30 d if `remember_me=true`).
- **Maximum 4 active sessions**; older ones are auto-revoked.
```

---

