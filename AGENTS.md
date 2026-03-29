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
- **Environment variables** for database DSN, port, and JWT secret (`.env` + Viper or plain `os.Getenv`).
- **Consistent HTTP error handling** — don't return `200` on failure, and don't leak error details in the body in production.
- Include a `docker-compose.yml` that spins up MySQL to ease evaluation.

---

## What Is NOT Required

- Frontend
- Deploy or CI/CD
- Pagination *(welcome, but optional)*
- Swagger *(welcome, but optional)*

---

## ⭐ Bonus — File Upload with Abstracted Storage

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

Implement two versions of this interface:

- **`LocalStorage`** — saves the file to local disk (for running without cloud)
- **`GCSStorage`** *(optional)* — saves to Google Cloud Storage using `cloud.google.com/go/storage`

Choose which implementation to use via environment variable (`STORAGE_BACKEND=local` or `gcs`).

**Validate:** only images (`.jpg`, `.png`, `.webp`), max 5 MB.

Write at least one test using the `LocalStorage` implementation or a mock of the interface.

### Evaluation Criteria

- Use of interface to decouple the handler from the concrete storage
- Dependency injection in the handler
- Error handling (invalid file, storage unavailable)
- Ability to mock `StorageProvider` in tests

---

## Delivery

Submit a link to a public repository (GitHub/GitLab) or a `.zip` containing:

- Source code
- Functional `docker-compose.yml`
- `README.md` with instructions to run locally
- `SOLUTION.md` with:
  - Design decisions you made
  - What you would do differently with more time
  - Trade-offs you identified