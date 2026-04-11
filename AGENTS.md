# AGENTS.md

## Tooling

### mise
[mise](https://mise.jdx.dev/) manages tool versions for this project. Versions are declared in `mise.toml`:
- `go` — Go version
- `task` — Task runner
- `air` — Live reload for development
- `golangci-lint` — Go linting used by `task lint`

Run `mise install` to install all required tools before working on this project, including `golangci-lint` for the default `task lint` verification step.

### Taskfile
[Task](https://taskfile.dev/) is used as the task runner (`Taskfile.yml`). Common tasks:

| Command      | Description                        |
|--------------|------------------------------------|
| `task dev`   | Start dev server with live reload (via `air`) |
| `task run`   | Run the application                |
| `task build` | Build binary to `bin/app`          |
| `task tidy`  | Tidy Go modules                    |
| `task lint`                  | Run Go linting checks (`golangci-lint`)                     |
| `task seed`                  | Seed DB with test data                                      |
| `task test:unit`             | Run unit tests and generate `unit_coverage.out`             |
| `task test:unit:coverage`    | Run unit tests and assert coverage ≥ 85% across all `internal/` packages |
| `task test:behavior`         | Run behavior/integration tests (spins up postgres via Docker) and generate `coverage.out` |
| `task test:behavior:coverage`| Run behavior tests and assert coverage ≥ 80%               |
| `task up`                    | Start full stack in Docker with live rebuild on changes      |
| `task behavior:diff`         | Show which behavior tests changed since last commit         |

## Verification

Run the narrowest checks that match the change, and mention anything you could not run.

- Default for any code change: `task lint` and `task test:unit:coverage`
- If you changed API handlers, database code, migrations, seeds, or integration test helpers: also run `task test:behavior:coverage`
- If you changed build wiring, CLI startup, or dependencies: also run `task build`
- If behavior tests changed through the behavior-test skill: `task behavior:diff` and `task test:behavior:coverage`

**When introducing new code paths** (new error-handling branches, new conditions, new response codes): add tests for those paths in the same change, before running coverage checks. Coverage checks are a final gate — they should confirm tests pass, not be how you discover that tests are missing.

- Add unit tests (in `internal/<domain>/handler_test.go`) for DB-error paths and other cases that require mocking.
- Add or update behavior tests via the `/behavior-test` skill for any new or changed observable API behavior (new query parameters, new success/error responses, new endpoints, or intentional behavior changes). Run this skill as part of the same change so both coverage gates pass together.

After all checks pass, run `/review` on the changed files before considering the task done. This catches issues that coverage checks won't — such as loose test assertions, missing mock expectations, non-deterministic ordering, or logic correctness problems.

## Behavior Tests

Integration tests that verify the observable behavior of the API live in `cmd/api/behavior/`:

- `todo_test.go` — `TestBehavior_Todo_*` functions
- `user_test.go` — `TestBehavior_User_*` functions

**Naming convention:** `TestBehavior_<Domain>_<Action>_<Expectation>`

Examples:
- `TestBehavior_Todo_List_ReturnsSeededTodos`
- `TestBehavior_Todo_Create_PersistsAndReturns`
- `TestBehavior_Todo_Get_Returns404ForUnknownID`

**Rule: behavior test case files in `cmd/api/behavior/` may need to change when observable API behavior changes, but do not modify them directly.** This applies to files such as `todo_test.go` and `user_test.go`. This repo expects those tests to be added or updated through the `/behavior-test` skill so naming and structure stay consistent. If that skill is unavailable in the current runtime, stop and tell the user instead of editing the test case files manually.

Test infrastructure (TestMain, helpers) lives in `cmd/api/behavior/testmain_test.go` and can be modified normally.

## Database

PostgreSQL 18.x is used locally and in tests. The current repo wiring pins `postgres:18.3` in Docker and integration tests. Schema is managed via [golang-migrate](https://github.com/golang-migrate/migrate) with embedded SQL files.

### Structure

- `internal/platform/db/migrations/` — schema migrations (`000001_create_todos.up.sql` / `.down.sql`). Auto-applied at startup.
- `internal/platform/db/seeds/001_todos.sql` — test data. Run with `task seed` (idempotent only if table is empty).

### Local development

Requires a local PostgreSQL instance. Copy `.env.example` to `.env` and adjust `DATABASE_URL` as needed:

```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/todos?sslmode=disable
```

Create the database before first run:
```
createdb todos
```

### Docker

`task up` starts both the API and a PostgreSQL 18.3 container with live rebuild on source changes. The API waits for the DB to be healthy before starting.

To seed in Docker:
```
docker compose run --rm api ./app -seed
```
