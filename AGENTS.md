# Technical Challenge

## Context

Build a mini REST API for team-based task management from scratch. The goal isn't to deliver a complete product, but to demonstrate how you structure and write code using the stack below:

| Technology | Version |
|---|---|
| Go | 1.26 |
| Echo | v5 |
| GORM | v2 |
| MySQL | 8.x |
| JWT | github.com/golang-jwt/jwt/v5 |
| Logger | go.uber.org/zap |

---

## Engineering Standards

## Go Code Conventions

Mandatory standards: [Google Go Style Guide](https://google.github.io/styleguide/go/) + [Effective Go](https://go.dev/doc/effective_go).

- **Package names:** lowercase single word, no underscores (`versioning`, not `document_versioning`)
- **No export stutter:** `verify.Result` ✓ — `verify.VerificationResult` ✗; `sharing.Token` ✓ — `sharing.SharingToken` ✗
- **Errors as values, defined in the owning slice:**
  `upload.ErrUnsupportedFormat`, `verify.ErrDivergent`, `verify.ErrAlgorithmUnsupported`,
  `sharing.ErrTokenExpired`, `sharing.ErrTokenRevoked`,
  `platform/session.ErrNotFound`, `platform/session.ErrInvalidSignature`
- **Error wrapping:** `fmt.Errorf("upload.service.Store: %w", err)` — always inspectable via `errors.Is`/`errors.As`
- **Interfaces at point of use, small and focused:**
  `verify.Verifier`, `platform/audit.Writer`, `platform/session.Store`, `sharing.TokenStore`
- **Constructor functions for DI:** `upload.NewService(querier, auditor)` — no package-level vars, no `init()`
- **`context.Context` as first param** in all service methods and repository functions; never stored in structs
- **Functional options** for optional config: `WithAlgorithm(a Algorithm)`
- **Godoc on every exported identifier** starting with the identifier name
- **`io.TeeReader`** for streaming hash computation — never buffer entire file in memory

### Project Structure — Vertical Slices
Code is organized by **feature (use case)**, not by technical layer. Each slice owns its handler, service logic, and types end-to-end. Cross-cutting infrastructure lives in `platform/`.

The rule: **if you're working on a feature, you should rarely need to leave its slice directory.**
```
internal/
  auth/
    handler.go
    service.go
    repository.go
    model.go
  teams/
    handler.go
    service.go
    repository.go
    model.go
  tasks/
    handler.go
    service.go
    repository.go
    model.go
  users/
    handler.go
    service.go
    repository.go
    model.go
```

Shared infrastructure (DB, logger, JWT middleware) lives in `internal/infra/` or `internal/middleware/`.

### Slice anatomy

Each feature slice follows the same internal shape:

```
upload/
  handler.go    # chi handler(s); reads request, calls service, renders templ
  service.go    # business logic; depends on store/db.Querier + platform/audit
  types.go      # request/response types and domain errors for this slice
                # e.g. upload.ErrUnsupportedFormat
```

Slices **do not import each other**. Shared types (e.g. a document ID passed between slices) come from `store/db` models or a minimal shared `types.go` at `internal/types.go` if truly needed.

### Platform packages

`platform/` packages are the only ones imported by multiple slices:


### Database Migrations
Use **GORM CLI** (`go run gorm.io/gorm/cmd/gorm`) for migrations. Do NOT use GORM Gen.

### Password Hashing
Use **bcrypt** with a cost factor of **12**.

### Payload Validation
Use **`github.com/go-playground/validator/v10`** for all request payload validation. Bind and validate in the handler before passing to the service layer.

### Testing
- Follow the **`/tdd-workflow`** skill: write tests first (RED → GREEN → REFACTOR).

#### Testing Philosophy

**Integration tests via testcontainers-go are the primary form of testing.** Each test package spins up a real MySQL container via `testhelper.NewMySQLContainer(t)`, runs all GORM auto-migrations, and exercises real behavior end-to-end. Each test runs inside a GORM transaction rolled back in `t.Cleanup` — no manual truncation.

Unit tests are reserved for pure logic with zero I/O: hash computation, JWT signing, constant-time comparisons, TTL math. **No DB mocks.**

**No testify.** All assertions use the standard `testing` package: `t.Fatal`, `t.Fatalf`, `t.Errorf`. No `require`, no `assert`.
```bash
go test -race -count=1 ./...   # -count=1 disables caching so containers always run fresh
```

### Pattern 1 — TestTx with t.Cleanup

`testhelper.TestTx` begins a GORM transaction and registers its rollback via `t.Cleanup`. No explicit `defer` in tests. Container is a singleton per test binary run (`sync.Once`).
```go
// internal/testhelper/testhelper.go

// NewMySQLContainer returns a *gorm.DB backed by a MySQL testcontainer.
// The container starts once per binary run; subsequent calls return the same instance.
func NewMySQLContainer(t testing.TB) *gorm.DB

// TestTx begins a *gorm.DB transaction and registers its rollback via t.Cleanup.
// Each test that calls TestTx gets a fully isolated transaction.
func TestTx(t testing.TB, db *gorm.DB) *gorm.DB
```

Implementation sketch:
```go
func TestTx(t testing.TB, db *gorm.DB) *gorm.DB {
    t.Helper()
    tx := db.Begin()
    if tx.Error != nil {
        t.Fatalf("testhelper: begin tx: %v", tx.Error)
    }
    t.Cleanup(func() {
        if err := tx.Rollback().Error; err != nil && !errors.Is(err, sql.ErrTxDone) {
            t.Errorf("testhelper: rollback tx: %v", err)
        }
    })
    return tx
}
```

### Pattern 2 — Parallel test bundle

Each `Test*` function defines a local `testBundle` struct and a `setup` closure. Every subtest calls `setup(t)` + `t.Parallel()`, getting its own isolated transaction.
```go
func TestTaskService(t *testing.T) {
    type testBundle struct {
        svc *Service
        tx  *gorm.DB
    }

    setup := func(t *testing.T) (*testBundle, context.Context) {
        t.Helper()
        ctx := context.Background()
        db := testhelper.NewMySQLContainer(t)
        tx := testhelper.TestTx(t, db)
        return &testBundle{
            svc: NewService(tx),
            tx:  tx,
        }, ctx
    }

    t.Run("completes task and updates score", func(t *testing.T) {
        t.Parallel()
        bundle, ctx := setup(t)
        // ... assertions using t.Fatal / t.Errorf only
    })

    t.Run("rejects duplicate completion", func(t *testing.T) {
        t.Parallel()
        bundle, ctx := setup(t)
        // ...
    })
}
```

Rules:
- Bundle fields are the service under test + its direct fixtures (no globals).
- `setup` always calls `t.Helper()` first.
- Subtests always call `t.Parallel()` immediately after receiving the bundle.

### Pattern 3 — Data fixtures (dbfactory)

`internal/testhelper/dbfactory` contains factory functions that insert real rows with sensible defaults using GORM. They call `t.Fatal` directly — no error return.
```go
// internal/testhelper/dbfactory/factory.go

type UserOpts struct {
    Email        string // generated if empty: "user-000001@example.com"
    PasswordHash string // generated if empty: bcrypt of "password"
    Score        int
}

func User(ctx context.Context, t *testing.T, tx *gorm.DB, opts *UserOpts) *model.User {
    t.Helper()
    if opts == nil {
        opts = &UserOpts{}
    }
    email := opts.Email
    if email == "" {
        email = fmt.Sprintf("user-%s@example.com", seqStr())
    }
    hash := opts.PasswordHash
    if hash == "" {
        b, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
        if err != nil {
            t.Fatalf("dbfactory.User: hash password: %v", err)
        }
        hash = string(b)
    }
    u := &model.User{Email: email, PasswordHash: hash, Score: opts.Score}
    if err := tx.WithContext(ctx).Create(u).Error; err != nil {
        t.Fatalf("dbfactory.User: %v", err)
    }
    return u
}
```

The atomic sequence counter ensures uniqueness across parallel tests:
```go
var seq atomic.Int64

func seqStr() string { return fmt.Sprintf("%06d", seq.Add(1)) }
```

Factories are grouped in `var` blocks at test sites — override only what matters:
```go
var (
    owner = dbfactory.User(ctx, t, tx, nil)
    team  = dbfactory.Team(ctx, t, tx, nil)
    task  = dbfactory.Task(ctx, t, tx, &dbfactory.TaskOpts{TeamID: team.ID})
)
```

---

## Domain

The API manages teams and their members, who can create and complete tasks. Each completed task earns points for the member.

**Minimum entities:**
```
User    { id, name, email, password_hash, score }
Team    { id, name }
Member  { id, user_id, team_id }   // a user can belong to multiple teams
Task    { id, title, points, team_id, done_by_user_id, done_at }
```

---

## Functional Requirements

### RF1 — Authentication

| Endpoint | Description |
|---|---|
| `POST /auth/register` | Creates a user. Password must be stored as a hash (bcrypt or similar). |
| `POST /auth/login` | Validates credentials and returns a JWT in the response body. |

All other endpoints require the JWT in the `Authorization: Bearer <token>` header.

### RF2 — Teams & Members

| Endpoint | Description |
|---|---|
| `POST /teams` | Creates a team |
| `POST /teams/:id/members` | Adds a user to the team |
| `GET /teams/:id/members` | Lists team members with each member's `score` |

### RF3 — Tasks

| Endpoint | Description |
|---|---|
| `POST /teams/:id/tasks` | Creates a task for the team |
| `PATCH /tasks/:id/complete` | Marks the task as completed by the authenticated user. Adds `points` to the user's `score`. |
| `GET /teams/:id/tasks` | Lists team tasks (with optional filter `?done=true/false`) |

**Business rules:**
- A task can only be completed once.
- Only members of the team can complete that team's tasks.
- The point addition to `score` must be atomic (use a transaction).

### RF4 — Ranking

| Endpoint | Description |
|---|---|
| `GET /teams/:id/ranking` | Returns members ordered by `score` |

---

## Non-Functional Requirements

- **Structured logs** — at minimum on errors and write operations.
- **Environment variables** for database DSN, port, and JWT secret (`.env` + godotenv or plain `os.Getenv`).
- **Consistent HTTP error handling** — don't return `200` on failure, and don't leak error details in the body in production.
- Include a `docker-compose.yml` that spins up MySQL to ease evaluation.
- Pagination
- Open API 3.1 spec, using redocly

---

## What Is NOT Required

- Frontend
- Deploy or CI/CD

---

## File Upload with Abstracted Storage

Add the ability to upload a profile picture to a user's profile.

**Endpoint:**
```
POST /users/me/avatar
Content-Type: multipart/form-data
```

Returns the public URL (or identifier) of the saved file.

### Requirements

Define a `StorageProvider` interface with at least:
```go
type StorageProvider interface {
    Upload(ctx context.Context, filename string, content io.Reader) (string, error)
    Delete(ctx context.Context, filename string) error
}
```

Implement one version of this interface:

- **`LocalStorage`** — saves the file to local disk (for running without cloud)

**Validate:** only images (`.jpg`, `.png`, `.webp`), max 5 MB, and use http.DetectContentType for security.

Write at least one test using the `LocalStorage` implementation or a mock of the interface.

### Evaluation Criteria

- Use of interface to decouple the handler from the concrete storage
- Dependency injection in the handler
- Error handling (invalid file, storage unavailable)
- Ability to mock `StorageProvider` in tests

---

## Delivery
- Source code
- Functional `docker-compose.yml`
- `README.md` with instructions to run locally
- `SOLUTION.md` with:
  - Design decisions you made
  - What you would do differently with more time
  - Trade-offs you identified