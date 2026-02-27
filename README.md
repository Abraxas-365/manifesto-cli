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
- **PostgreSQL + Redis** setup with connection pooling
- **Docker Compose** and a full **Makefile** with 40+ commands (dev, build, test, migrate, backup, etc.)
- **Structured logging** with colored console output and JSON formatters
- **Rich error handling** via `errx` — typed errors with HTTP status codes, error registries, and context

Core libraries are always present. Everything else — file storage, async jobs, IAM, AI — is added on demand via `manifesto add`.

## Core Libraries

| Library | What it provides |
|---------|-----------------|
| `kernel` | Domain primitives, typed IDs, pagination, auth context |
| `errx` | Structured errors with HTTP status mapping and registries |
| `logx` | Colored console logging, JSON formatters, structured fields |
| `ptrx` | Pointer utility helpers |
| `config` | Environment-driven configuration with defaults |

## Modules

Modules are downloaded and wired into your project's container, config, server, and Makefile via code injection at marker points. Add them during init or later with `manifesto add`:

| Module | What it provides |
|--------|-----------------|
| `fsx` | File system abstraction — local disk or S3 with storage config |
| `asyncx` | Futures, fan-out, pools, retry, timeout patterns |
| `ai` | LLM clients, embeddings, vector store, OCR, speech (requires fsx) |
| `jobx` | Async job queue — Redis-backed dispatcher (requires asyncx) |
| `notifx` | Email notifications — SES notifier with email config |
| `iam` | Full auth system — OAuth, passwordless OTP, JWT, API keys, RBAC, multi-tenant users, sessions, invitations |

**Dependencies are resolved automatically:** `manifesto add jobx` downloads both `asyncx` and `jobx`. `manifesto add ai` downloads both `fsx` and `ai`.

**Cross-module bridges:** when both `jobx` and `notifx` are wired, the `notifx:send_email` async handler is automatically registered with the dispatcher.

## Usage

### Create a new project

```bash
# Interactive — prompts which modules to wire
manifesto init myapp --module github.com/me/myapp

# Wire specific modules during init
manifesto init myapp --module github.com/me/myapp --with fsx,jobx,iam

# Wire everything
manifesto init myapp --module github.com/me/myapp --all
```

### Create a quick project

Use `--quick` for a lightweight project without IAM or migrations:

```bash
manifesto init myapp --module github.com/me/myapp --quick

# Quick + wire fsx and jobx
manifesto init myapp --module github.com/me/myapp --quick --with fsx,jobx
```

Quick projects pull from the [`quick-project`](https://github.com/Abraxas-365/manifesto/tree/quick-project) branch and exclude IAM from the available modules.

### Add a module

```bash
cd myapp
manifesto add fsx       # Downloads fsx + wires file storage into container
manifesto add jobx      # Downloads asyncx + jobx, wires dispatcher
manifesto add notifx    # Wires notifx + detects jobx → injects bridge
manifesto add iam       # Downloads iam + migrations, wires into container + server + config
```

Adding is idempotent — running `manifesto add jobx` twice is a no-op.

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
  Core Libraries

    ● kernel       Domain primitives, value objects, pagination
    ● errx         Structured error handling with HTTP mapping
    ● logx         Structured logging (console/JSON)
    ● config       Environment-driven configuration
    ...

  Wireable Modules

    ● wired    fsx       File system abstraction (local, S3)
    ● wired    jobx      Async job queue (Redis-backed dispatcher)
    ○ not wired notifx   Email notifications (AWS SES)
    ○ not wired iam      Auth, users, tenants, scopes, API keys
```

## How Wiring Works

When you run `manifesto add <module>`, the CLI:

1. **Downloads** the module's source code from GitHub (if not already present)
2. **Resolves dependencies** — `jobx` auto-downloads `asyncx`, `ai` auto-downloads `fsx`
3. **Injects code** into your project files at marker comments
4. **Installs Go dependencies** (e.g., AWS SDK for fsx/notifx)
5. **Updates manifesto.yaml** to track wired modules

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
| `Makefile` | `# manifesto:env-config` | Environment variables |
| `Makefile` | `# manifesto:env-display` | `make env` display lines |

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
│   └── ptrx/               # Pointer utilities
├── docker-compose.yml      # Postgres + Redis
├── Makefile                # 40+ commands, all env vars
└── manifesto.yaml          # Project manifest (tracks wired modules)
```

After wiring modules, additional directories appear in `pkg/`:

```
├── pkg/
│   ├── fsx/                # File system abstraction (after: manifesto add fsx)
│   ├── asyncx/             # Async primitives (after: manifesto add asyncx)
│   ├── ai/                 # AI/LLM toolkit (after: manifesto add ai)
│   ├── jobx/               # Job queue (after: manifesto add jobx)
│   ├── notifx/             # Notifications (after: manifesto add notifx)
│   └── iam/                # IAM (after: manifesto add iam)
├── migrations/             # SQL migrations (after: manifesto add iam)
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
| `manifesto add <module>` | Add a module (fsx, asyncx, ai, jobx, notifx, iam) |
| `manifesto add <path>` | Add a DDD domain package |
| `manifesto modules` | List all libraries and modules |
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
