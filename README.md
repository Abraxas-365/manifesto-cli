# Manifesto CLI

Scaffold production-grade Go applications with DDD architecture, multi-tenancy, and modular design — in seconds.

Manifesto CLI is the companion tool for the [Manifesto Architecture](https://github.com/Abraxas-365/manifesto), a battle-tested Go project skeleton with built-in patterns for Domain-Driven Design, layered architecture, and enterprise-grade IAM.

## Install

```bash
go install github.com/Abraxas-365/manifesto-cli/cmd/manifesto@latest
```

Requires Go 1.23+.

## What You Get

Running `manifesto init` generates a complete, compilable Go project with:

- **Fiber HTTP server** with structured error handling, CORS, request IDs, and graceful shutdown
- **Dependency injection container** wired and ready
- **PostgreSQL + Redis** setup with connection pooling and migration scaffolding
- **File storage** abstraction (local disk or S3, swappable via env var)
- **Docker Compose** and a full **Makefile** with 40+ commands (dev, build, test, migrate, backup, etc.)
- **Structured logging** with colored console output and JSON formatters
- **Rich error handling** via `errx` — typed errors with HTTP status codes, error registries, and context

All library code lives in `pkg/` and is always present. Modules like IAM, job processing, and notifications are **dormant until wired** — no conditional templates, no code bloat.

## Libraries (always present)

| Library | What it provides |
|---------|-----------------|
| `kernel` | Domain primitives, typed IDs, pagination, auth context |
| `errx` | Structured errors with HTTP status mapping and registries |
| `logx` | Colored console logging, JSON formatters, structured fields |
| `ptrx` | Pointer utility helpers |
| `asyncx` | Futures, fan-out, pools, retry, timeout patterns |
| `config` | Environment-driven configuration with defaults |
| `fsx` | File system abstraction — local disk or S3 |
| `ai` | LLM clients, embeddings, vector store, OCR, speech |

## Wireable Modules

Modules are wired into your project's container, config, and server via code injection at marker points. Wire them during init or add them later:

| Module | What it wires |
|--------|--------------|
| `jobx` | Async job queue — Redis-backed dispatcher in container, background service startup |
| `notifx` | Email notifications — SES notifier in container, email config |
| `iam` | Full auth system — OAuth, passwordless OTP, JWT, API keys, RBAC, multi-tenant users, sessions, invitation-only registration |

**Cross-module bridges:** when both `jobx` and `notifx` are wired, the `notifx:send_email` async handler is automatically registered with the dispatcher.

## Usage

### Create a new project

```bash
# Interactive — prompts which modules to wire
manifesto init myapp --module github.com/me/myapp

# Wire specific modules during init
manifesto init myapp --module github.com/me/myapp --with jobx,iam

# Wire everything
manifesto init myapp --module github.com/me/myapp --all
```

### Create a quick project

Use `--quick` for a lightweight project without IAM or migrations:

```bash
manifesto init myapp --module github.com/me/myapp --quick

# Quick + wire jobx
manifesto init myapp --module github.com/me/myapp --quick --with jobx
```

Quick projects pull from the [`quick-project`](https://github.com/Abraxas-365/manifesto/tree/quick-project) branch and exclude IAM and migrations from the download.

### Wire a module

```bash
cd myapp
manifesto add jobx      # Wires jobx into container + config
manifesto add notifx    # Wires notifx + detects jobx → injects bridge
manifesto add iam       # Wires iam into container + server + config
```

Wiring is idempotent — running `manifesto add jobx` twice is a no-op.

### Add a domain package

```bash
manifesto add pkg/recruitment/candidate
```

This generates a full DDD domain vertical:

```
pkg/recruitment/candidate/
├── candidate.go              # Entity + domain methods + DTOs
├── port.go                   # Repository interface
├── errors.go                 # Error registry (errx)
├── candidatesrv/
│   └── service.go            # Business logic layer
├── candidateinfra/
│   └── postgres.go           # PostgreSQL repository
├── candidateapi/
│   └── handler.go            # Fiber HTTP handlers
└── candidatecontainer/
    └── container.go          # Module DI wiring
```

Plus a typed ID appended to `pkg/kernel/ids.go` and automatic injection into `cmd/container.go` and `cmd/server.go`.

### List modules

```bash
manifesto modules
```

```
  Libraries (always present)

    ● kernel       Domain primitives, value objects, pagination
    ● errx         Structured error handling with HTTP mapping
    ● logx         Structured logging (console/JSON)
    ● config       Environment-driven configuration
    ● fsx          File system abstraction (local, S3)
    ...

  Wireable Modules

    ● wired    jobx      Async job queue (Redis-backed dispatcher)
    ○ not wired notifx   Email notifications (AWS SES)
    ○ not wired iam      Auth, users, tenants, scopes, API keys
```

## How Wiring Works

All module code lives in `pkg/` from day one. Wiring injects references into your project files at marker comments:

| File | Marker | Purpose |
|------|--------|---------|
| `pkg/config/config.go` | `// manifesto:config-fields` | Config struct fields |
| `pkg/config/config.go` | `// manifesto:config-loads` | Load() assignments |
| `cmd/container.go` | `// manifesto:container-imports` | Import lines |
| `cmd/container.go` | `// manifesto:container-fields` | Struct fields |
| `cmd/container.go` | `// manifesto:module-init` | initModules() code |
| `cmd/container.go` | `// manifesto:background-start` | Background services |
| `cmd/container.go` | `// manifesto:container-helpers` | Top-level functions |
| `cmd/server.go` | `// manifesto:server-imports` | Import lines |
| `cmd/server.go` | `// manifesto:public-routes` | Public routes (OAuth) |
| `cmd/server.go` | `// manifesto:route-registration` | Protected routes |

The same marker system is used by `manifesto add <domain-path>` to inject domain containers and routes.

## Generated Project Structure

```
myapp/
├── cmd/
│   ├── server.go           # Fiber app, middleware, routes, graceful shutdown
│   └── container.go        # Dependency injection, wiring
├── pkg/
│   ├── kernel/             # Shared types: IDs, pagination, auth context
│   ├── errx/               # Structured errors with HTTP mapping
│   ├── logx/               # Logging framework
│   ├── config/             # Env-driven config loading
│   ├── ptrx/               # Pointer utilities
│   ├── asyncx/             # Async primitives
│   ├── fsx/                # File system abstraction
│   ├── ai/                 # AI/LLM toolkit
│   ├── jobx/               # Job queue (dormant until wired)
│   ├── notifx/             # Notifications (dormant until wired)
│   └── iam/                # IAM (dormant until wired)
├── migrations/             # SQL migration files
├── docker-compose.yml      # Postgres + Redis
├── Makefile                # 40+ commands, all env vars
└── manifesto.yaml          # Project manifest (tracks wired modules)
```

## Architecture

Manifesto follows a strict layered DDD architecture. Each domain is self-contained and independent:

```
  API Layer (handlers)          ← HTTP boundary
  Service Layer (business)      ← Orchestration
  Domain Layer (entities)       ← Core rules, zero dependencies
  Repository Layer (ports)      ← Contracts
  Infrastructure (postgres)     ← Implementation details
```

**Key rules:**

- Dependencies flow downward only — no cycles
- Domain layer has zero external imports
- Repository interfaces live in the domain, implementations in `*infra/`
- Services orchestrate cross-domain logic; entities enforce rules
- Domains reference each other only through kernel IDs, never direct imports

For the full architecture guide, patterns, and rationale, see the [Manifesto Architecture Document](https://github.com/Abraxas-365/manifesto).

## Commands

| Command | Description |
|---------|-------------|
| `manifesto init <name> --module <go-module>` | Create a new project |
| `manifesto add <module>` | Wire a module (jobx, notifx, iam) |
| `manifesto add <path>` | Add a DDD domain package |
| `manifesto modules` | List all libraries and wireable modules |
| `manifesto version` | Show CLI version |

### Flags

| Flag | Used with | Description |
|------|-----------|-------------|
| `--with <modules>` | `init` | Comma-separated modules to wire |
| `--all` | `init` | Wire all available modules |
| `--quick` | `init` | Lightweight project (no IAM, no migrations) |
| `--ref <version>` | `init` | Pin manifesto version (default: latest) |

## Generated Makefile Commands

The generated Makefile includes everything you need:

```bash
make dev              # Run development server
make dev-watch        # Hot reload with air
make build            # Build binary
make test             # Run tests
make lint             # golangci-lint

make up               # Start all Docker services
make down             # Stop services
make health           # Check service health

make migrate          # Run migrations
make migrate-create   # Create new migration file
make db-reset         # Clean + migrate + seed
make db-backup        # Backup to file

make psql             # Open psql shell
make redis-cli        # Open Redis CLI
make env              # Show all config
```

## License

MIT
