# Mnemosyne — Backend

Collaborative note editor backend using **Operational Transformation (OT)** for real-time sync. Built with Go, Chi, PostgreSQL, and WebSockets.

---

## Stack

| Layer | Tool |
|---|---|
| Framework | [Chi](https://github.com/go-chi/chi) |
| Database | PostgreSQL |
| Migrations | [Goose](https://github.com/pressly/goose) |
| Query generation | [SQLc](https://sqlc.dev) |
| Real-time sync | WebSockets + OT (EasySync) |
| Auth | JWT (httpOnly cookies) |

---

## Prerequisites

- **Go** (1.21+)
- **PostgreSQL** (running locally or accessible remotely)
- **Goose** — `go install github.com/pressly/goose/v3/cmd/goose@latest`
- **SQLc** — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

---

## Getting Started

### 1. Configure environment

```env
PORT=8080
SECRET_KEY=<generate a random key>
BLUEPRINT_DB_HOST=localhost
BLUEPRINT_DB_PORT=5432
BLUEPRINT_DB_USERNAME=your_user
BLUEPRINT_DB_PASSWORD=your_password
BLUEPRINT_DB_DATABASE=your_db
BLUEPRINT_DB_SCHEMA=public
```

### 2. Run migrations

```bash
goose up
```

### 3. Start the server

```bash
go run cmd/main.go
```

---

## OT Implementation

The collaborative editing backend uses **Operational Transformation (OT)**. The implementation is based on the EasySync algorithm described in the following paper:

- [EasySync: A Comprehensive Description of the EasySync OT Protocol](https://github.com/knemerzitski/notes/blob/article/packages/collab/docs/easysync-full-description.pdf)

This document served as the primary reference for the OT protocol, including document model, operation types, transformation functions, and merge logic.

---

- Database connection reads from `BLUEPRINT_DB_*` env vars
- The `Makefile` includes commands for `goose` and `sqlc` workflows