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
- **PostgreSQL** setup with sqlx, connection pooling, and migration scaffolding
- **Docker Compose** and a full **Makefile** with 40+ commands (dev, build, test, migrate, backup, etc.)
- **Structured logging** with colored console output and JSON formatters
- **Rich error handling** via `errx` — typed errors with HTTP status codes, error registries, and context

Add modules for more:

| Module | What it adds |
|--------|-------------|
| `iam` | OAuth (Google/Microsoft), passwordless OTP, JWT, API keys, RBAC scopes, multi-tenant users, invitations, Redis sessions |
| `fsx` | File system abstraction — swap between local disk and S3 with one env var |
| `ai` | LLM clients, embeddings, vector store, OCR, speech-to-text |

Everything is **conditional**. No IAM? No Redis in your docker-compose, no auth middleware in your server, no OAuth env vars in your Makefile.

## Usage

### Create a new project

```bash
# Interactive — core modules only
manifesto init myapp --module github.com/me/myapp

# With specific modules
manifesto init myapp --module github.com/me/myapp --with iam,fsx

# Everything
manifesto init myapp --module github.com/me/myapp --all
```

### Create a quick project

Use `--quick` for a lightweight project without IAM or migrations. You still get the same interactive selection for optional modules (`fsx`, `asyncx`, `ai`).

```bash
# Interactive — pick optional modules
manifesto init myapp --module github.com/me/myapp --quick

# With specific modules
manifesto init myapp --module github.com/me/myapp --quick --with fsx,asyncx

# All optional modules
manifesto init myapp --module github.com/me/myapp --quick --all
```

Quick projects include a smaller set of core modules (`kernel`, `errx`, `logx`, `ptrx`, `config`, `server`) and pull from the [`quick-project`](https://github.com/Abraxas-365/manifesto/tree/quick-project) branch.

### Add a domain package

```bash
cd myapp
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
└── candidateapi/
    └── handler.go            # Fiber HTTP handlers
```

Plus a typed ID appended to `pkg/kernel/ids.go`:

```go
type CandidateID string
func NewCandidateID(id string) CandidateID { return CandidateID(id) }
func (id CandidateID) String() string      { return string(id) }
func (id CandidateID) IsEmpty() bool       { return string(id) == "" }
```

### Install a module later

```bash
manifesto install ai    # Also installs fsx (dependency)
```

### List available modules

```bash
manifesto modules
```

```
  ● kernel       Domain primitives, value objects, pagination
  ● errx         Structured error handling with HTTP mapping
  ● logx         Structured logging (console/JSON)
  ● config       Environment-driven configuration
  ○ iam          Auth, users, tenants, scopes, API keys
  ○ fsx          File system abstraction (local, S3)
  ○ ai           LLM, embeddings, vector store, OCR, speech

  ● installed  ○ available
```

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
│   ├── iam/                # (if installed) Full IAM module
│   │   ├── auth/           #   OAuth, JWT, middleware, scopes
│   │   ├── user/           #   User entity + repo + service
│   │   ├── tenant/         #   Multi-tenant management
│   │   ├── apikey/         #   API key management
│   │   ├── invitation/     #   Invitation-only registration
│   │   └── otp/            #   Passwordless OTP auth
│   ├── fsx/                # (if installed) File system abstraction
│   └── ai/                 # (if installed) AI/LLM toolkit
├── migrations/             # SQL migration files
├── docker-compose.yml      # Postgres + Redis (if IAM)
├── Makefile                # 40+ commands, all env vars
└── manifesto.yaml          # Project manifest
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

## Module Details

### Core (always included)

**kernel** — Shared domain primitives: typed IDs (`UserID`, `TenantID`), `Paginated[T]` generic container, `PaginationOptions`, `AuthContext`.

**errx** — Error registries per module, typed errors (`TypeValidation`, `TypeBusiness`, `TypeInternal`), automatic HTTP status mapping, `WithDetail()` for context, `Wrap()` for chains.

**logx** — Rust-inspired colored console logging, JSON/CloudWatch formatters, structured fields, level-based filtering.

**config** — Environment-driven configuration with defaults, validation on startup, `IsDevelopment()` helpers.

### Optional

**iam** — Complete identity and access management: OAuth 2.0 (Google, Microsoft), passwordless OTP, JWT access/refresh tokens, API key management with live/test modes, scope-based RBAC with wildcard matching, multi-tenant isolation, invitation-only registration, session management with background cleanup.

**fsx** — File system abstraction with `fsx.FileSystem` interface, local disk and S3 implementations, swap via `STORAGE_MODE` env var.

**ai** — LLM client abstraction, embeddings, vector store integration, OCR, speech-to-text. Requires `fsx`.

## Commands

| Command | Description |
|---------|-------------|
| `manifesto init <name> --module <go-module>` | Create a new project |
| `manifesto add <path>` | Add a DDD domain package |
| `manifesto install <module>` | Install a module (with deps) |
| `manifesto modules` | List all modules and status |
| `manifesto version` | Show CLI version |

### Flags

| Flag | Used with | Description |
|------|-----------|-------------|
| `--with <modules>` | `init` | Comma-separated optional modules |
| `--all` | `init` | Install all modules |
| `--quick` | `init` | Lightweight project (no IAM, no migrations) |
| `--ref <version>` | `init`, `install` | Pin manifesto version (default: latest) |

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
make redis-cli        # Open Redis CLI (if IAM)
make env              # Show all config
```

## License

MIT
