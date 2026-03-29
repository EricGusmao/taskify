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

You have full freedom to structure the project however you prefer.

---

## Engineering Standards

### Code Style
- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Google Go Style Guide](https://google.github.io/styleguide/go/).

### Project Structure — Vertical Slices
Organize code by feature/domain slice, not by technical layer. Each slice owns its handler, service, repository, and model:

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

### Database Migrations
Use **GORM CLI** (`go run gorm.io/gorm/cmd/gorm`) for migrations. Do NOT use GORM Gen.

### Password Hashing
Use **bcrypt** with a cost factor of **12**.

### Payload Validation
Use **`github.com/go-playground/validator/v10`** for all request payload validation. Bind and validate in the handler before passing to the service layer.

### Testing
- Follow the **`/tdd-workflow`** skill: write tests first (RED → GREEN → REFACTOR).
- Use **Testcontainers** (`github.com/testcontainers/testcontainers-go`) to spin up real MySQL containers in integration tests — no mocks for the database layer.

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