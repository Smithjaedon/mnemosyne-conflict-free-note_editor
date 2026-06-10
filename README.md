# goproto

A CLI tool for scaffolding production-ready Go web applications. `goproto` wraps [Go Blueprint](https://github.com/Melkeydev/go-blueprint) and layers in opinionated defaults — JWT authentication with httpOnly cookies, Goose migrations, and SQLc for type-safe database access — so you can skip the boilerplate and start building.

> This README covers the **minimal** branch, which uses a fixed, opinionated stack rather than the configurable options in the default branch.

---

## Stack

| Layer | Tool |
|---|---|
| Framework | [Chi](https://github.com/go-chi/chi) |
| Database | PostgreSQL |
| Migrations | [Goose](https://github.com/pressly/goose) |
| Query generation | [SQLc](https://sqlc.dev) |
| Live reload | [Air](https://github.com/air-verse/air) |
| Auth | JWT (httpOnly cookies) |

---

## Prerequisites

Before using `goproto`, make sure you have the following installed on your machine:

- **Go** (1.21+)
- **PostgreSQL** (running locally or accessible remotely)
- **Goose** — `go install github.com/pressly/goose/v3/cmd/goose@latest`
- **SQLc** — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- **Air** — `go install github.com/air-verse/air@latest`

---

## Getting Started

### 1. Create a new project

```bash
goproto create
```

This scaffolds the full project structure, including:

- Chi router setup
- JWT authentication middleware with httpOnly cookie handling
- A `users` migration as your first table
- SQLc `query/` and `schema/` directories, pre-organized by resource
- A `Makefile` with useful development commands
- A `.env` file with a freshly generated JWT secret key

### 2. Configure your environment

Open the generated `.env` file and update the following to match your setup:

```env
# Database connection variables — update to match your PostgreSQL instance
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_NAME=your_db

# Goose database string — must match your database credentials
GOOSE_DBSTRING=postgres://your_user:your_password@localhost:5432/your_db

# Server port — change if needed
PORT=8080

# JWT secret — auto-generated, leave this as-is
JWT_SECRET=<generated>
```

> Every project gets a unique JWT secret at generation time. Do not reuse secrets across projects.

### 3. Run your migrations

```bash
goose up
```

This creates your `users` table (and any other migrations you've added) in your database.

### 4. Start the development server

```bash
make watch
```

Your server is now running with live reload via Air.

---

## Project Structure

```
your-project/
├── migrations/          # Goose migration files
├── sqlc/
│   ├── query/           # Raw SQL queries, one file per resource
│   └── schema/          # Table schema definitions, one file per resource
├── .env                 # Environment variables
├── Makefile
└── ...
```

### Working with SQLc

SQLc reads your SQL files and generates type-safe Go code from them. The `query/` and `schema/` directories are organized by resource — so if you have users and notes, you get separate files for each. This keeps things readable as your project grows.

**Adding a new resource:**
1. Create a schema file in `sqlc/schema/` (e.g. `notes.sql`)
2. Create a query file in `sqlc/query/` (e.g. `notes.sql`)
3. Run `make generate` to regenerate Go code

### Working with Goose

Migrations live in the `migrations/` folder. The `users` table migration is created for you — edit it if you need to add or change user fields before your first `goose up`.

**Adding a new migration:**
```bash
make goose-create
```
You'll be prompted for a migration name (e.g. `add_notes`). Goose creates the file in `migrations/` — open it and write your SQL.

```bash
goose up      # apply all pending migrations
goose down    # roll back all migrations
```

---

## Makefile Commands

| Command | Description |
|---|---|
| `make watch` | Start the dev server with live reload (Air) |
| `make generate` | Run `sqlc generate` from the root directory |
| `make goose-create` | Prompt for a name and create a new migration file |

---

## Notes

- This tool is a wrapper around [Go Blueprint](https://github.com/Melkeydev/go-blueprint). The underlying project structure and some generated code come from Go Blueprint; `goproto` adds the auth setup, SQLc configuration, and Makefile commands on top.
- Docker support is included in the scaffolded project but is not covered in this guide.
- The **minimal** branch is opinionated by design. For a more flexible setup with configurable options, see the default branch.