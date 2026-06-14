# Mnemosyne — Backend

Real-time collaborative note editor backend. Built from scratch in Go — WebSocket sync, Operational Transformation, PostgreSQL persistence, and JWT auth.

---

## Architecture

```
Client ──WebSocket──►  Hub (connection manager)
                            │
                     ┌──────┴──────┐
                     │             │
                  OT Engine     Broadcast
                     │             │
                     └──────┬──────┘
                            │
                     PostgreSQL (sqlc)
```

### Layers

**WebSocket Layer** — Each client connects via a WebSocket. The hub tracks all active connections per document, routes messages to the OT engine, and broadcasts accepted operations to all other clients.

**OT Engine** — Implements the EasySync protocol (client-server OT). Operations are transformed against concurrent edits before being applied and broadcast. The engine handles:
- Insert and delete operations on a character-level document model
- Operation transformation against history buffers
- Acknowledgment tracking per client
- Undo support via operation metadata

**Persistence** — Notes are stored in PostgreSQL. The document state is rebuilt on server start by replaying the stored operation log. Migrations are managed with Goose.

**Auth** — JWT-based authentication with httpOnly cookies. Middleware protects all routes except signup/login.

---

## Stack

| Layer | Choice | Why |
|---|---|---|
| Language | Go 1.21+ | Concurrency model fits WebSocket fan-out; fast compile; deploys as a single binary |
| Router | [Chi](https://github.com/go-chi/chi) | Lightweight, stdlib-compatible, middleware chaining |
| Database | PostgreSQL | Reliable, good JSON support, well-understood |
| Migrations | [Goose](https://github.com/pressly/goose) | Version-controlled SQL migrations, easy rollbacks |
| Query gen | [SQLc](https://sqlc.dev) | Type-safe Go from raw SQL — no ORM overhead |
| Real-time | WebSockets (gorilla/websocket) | Bidirectional, low-latency, supported by all browsers |
| OT | EasySync protocol | Chosen over CRDTs for simpler server-authoritative model; see below |
| Auth | JWT (httpOnly cookies) | Stateless, secure against XSS, simple revocation via short expiry |

---

## OT vs CRDT — Why OT

This backend implements **Operational Transformation** rather than a CRDT. Both approaches solve the same problem (concurrent editing), but OT was chosen for this project because:

| Factor | OT | CRDT |
|---|---|---|
| Server model | Server-authoritative — server resolves conflicts | Peer-to-peer or server-as-relay |
| Complexity | Simple transform functions | Complex merge rules (RGA, YATA, etc.) |
| State size | Operation log (compact) | Full document metadata per node |
| Undo | Natural — revert by inverse operation | Requires tombstone/version vector management |
| Client sync | Acknowledgment-based; client waits for server confirmation | Optimistic; peers apply locally and merge later |

The **EasySync** protocol is a well-documented OT variant designed for text editing. It defines a small set of operation types (insert, delete) and a single transform function that resolves ordering conflicts deterministically.

---

## API

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/api/signup` | POST | No | Create account |
| `/api/login` | POST | No | Authenticate, receive cookie |
| `/api/notes` | GET | Yes | List user's notes |
| `/api/notes` | POST | Yes | Create a note |
| `/ws/note/{id}` | GET (upgrade) | Yes | WebSocket for real-time editing |

---

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL
- Goose — `go install github.com/pressly/goose/v3/cmd/goose@latest`
- SQLc — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

### 1. Configure environment

```env
PORT=8080
SECRET_KEY=<random 32+ char string>
BLUEPRINT_DB_HOST=localhost
BLUEPRINT_DB_PORT=5432
BLUEPRINT_DB_USERNAME=postgres
BLUEPRINT_DB_PASSWORD=postgres
BLUEPRINT_DB_DATABASE=mnemosyne
BLUEPRINT_DB_SCHEMA=public
```

### 2. Create database

```bash
createdb mnemosyne
```

### 3. Run migrations

```bash
goose up
```

### 4. Start server

```bash
go run cmd/main.go
```

---

## Project Structure

```
cmd/
  main.go              # Entry point, server bootstrap
internal/
  database/
    database.go        # Connection pool setup
  middleware/
    auth.go            # JWT middleware
  server/
    routes.go          # HTTP + WebSocket route registration
    ws.go              # WebSocket hub, connection management, message routing
    ot.go              # OT engine: operation types, transform, document model
    ot_test.go         # OT unit tests
    server.go          # HTTP server setup
db/
  models.go            # sqlc-generated Go types
  notes.sql.go         # sqlc-generated query methods
migrations/            # Goose SQL migrations
sqlc/
  queries/             # Raw SQL queries (sqlc input)
  schemas/             # Table definitions (sqlc input)
Makefile               # goose + sqlc task runner
```

---

## OT Implementation Details

The OT engine (in `internal/server/ot.go`) implements the EasySync protocol:

- **Operation types** — `Insert{Position, Text}` and `Delete{Position, Length}`. Operations are serialized as JSON for WebSocket transport.
- **Document model** — A flat character array. Every insert/delete is expressed as a position and content range.
- **Transformation** — When two operations conflict (same position, overlapping range), the transform function adjusts the later operation so it applies correctly against the already-applied state.
- **History buffer** — Each document maintains a list of acknowledged operations. New operations are transformed against the buffer before being applied.
- **Acknowledgment** — Clients track which operations they've sent. The server responds with an `ack` containing the operation's ID, allowing the client to update its local state and remove the op from its pending buffer.
- **Undo** — Each operation stores its inverse. An undo request sends the inverse operation through the same transform pipeline.

### Reference

- [EasySync: A Comprehensive Description of the EasySync OT Protocol](https://github.com/knemerzitski/notes/blob/article/packages/collab/docs/easysync-full-description.pdf)

---

## Deployment (Render)

| Env Variable | Value |
|---|---|
| `PORT` | `10000` |
| `SECRET_KEY` | Generate with `openssl rand -hex 32` |
| `BLUEPRINT_DB_HOST` | Render PostgreSQL internal hostname |
| `BLUEPRINT_DB_PORT` | `5432` |
| `BLUEPRINT_DB_USERNAME` | Render PostgreSQL user |
| `BLUEPRINT_DB_PASSWORD` | Render PostgreSQL password |
| `BLUEPRINT_DB_DATABASE` | Render PostgreSQL database name |
| `BLUEPRINT_DB_SCHEMA` | `public` |
