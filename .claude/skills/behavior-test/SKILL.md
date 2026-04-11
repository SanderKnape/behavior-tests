---
name: behavior-test
description: Review requests for behavior tests in this Go API, inspect the relevant handlers and existing behavior coverage, and route case-file changes through the required `/behavior-test` workflow instead of editing `cmd/api/behavior/todo_test.go` or `cmd/api/behavior/user_test.go` directly. Use for new observable API behavior, changed responses, new query parameters, or requests to add or update behavior tests. Do not use for unit-test-only work, DB failure-path tests, or direct edits to behavior test case files.
---

# Behavior Test Writer

Use this skill when the user wants to add, update, or review behavior tests for the API.

This repository treats behavior tests as the specification for observable API behavior, but behavior test case files must be updated through this `/behavior-test` skill.

## Your job

For each request:

1. Determine the observable behavior that needs coverage.
2. Inspect the relevant implementation and existing tests.
3. Decide whether the request belongs in behavior tests, unit tests, or both.
4. Route behavior test case changes through `/behavior-test` instead of editing case files directly.
5. If `/behavior-test` is unavailable in the current runtime, stop and tell the user clearly.

## What You May And May Not Edit

Do not directly edit behavior test case files in `cmd/api/behavior/`, including:

- `cmd/api/behavior/todo_test.go`
- `cmd/api/behavior/user_test.go`

Those files are maintained through the `/behavior-test` workflow.

You may inspect:

- `cmd/api/behavior/todo_test.go`
- `cmd/api/behavior/user_test.go`
- `cmd/api/behavior/testmain_test.go`
- relevant handlers, routes, DB code, migrations, and recent diffs

You may modify `cmd/api/behavior/testmain_test.go` only when the user specifically needs shared behavior-test infrastructure changes and that work is necessary for the requested behavior coverage.

## Read Order

Read only the files needed for the request.

1. Read the matching behavior test file:
   - todo-related requests: `cmd/api/behavior/todo_test.go`
   - user-related requests: `cmd/api/behavior/user_test.go`
2. Read `cmd/api/behavior/testmain_test.go` if you need helpers or test-environment details.
3. Read the relevant handler, route, persistence code, or diff so the requested behavior matches what the API actually does or should do.
4. Read the other domain's behavior file only if the request spans both resources or the correct location is unclear.

## When To Use Behavior Tests

Behavior tests are for observable API behavior such as:

- successful creates, reads, updates, and deletes
- validation failures
- `404` responses for missing resources
- list or query parameter behavior
- cross-resource flows that depend on real persistence or seeded data

Use unit tests instead for cases that depend on mocking or unusual DB behavior, such as:

- DB failure paths
- internal branching that is not primarily user-visible
- cases that require `sqlmock` or direct repository mocking

When new observable behavior is added, expect both:

- behavior coverage for the external API contract
- unit coverage for internal branches and DB-error paths where applicable

## Required Behavior-Test Conventions

### Naming

Use this naming convention when preparing or reviewing requested behavior tests:

```go
TestBehavior_<Domain>_<Action>_<Expectation>
```

Examples:

```go
TestBehavior_Todo_List_ReturnsSeededTodos
TestBehavior_Todo_Create_PersistsAndReturns
TestBehavior_User_Delete_Returns409WhenUserHasTodos
```

### Test Model

This repo's behavior tests use:

- `t.Parallel()` at the start of each test
- `env := newTestEnv(t)` for an isolated transaction-backed test environment
- `env.doRequest(...)` for HTTP requests
- helpers such as `createTodo`, `createUser`, and `decode`

When reviewing requested behavior coverage, make sure it follows the current helper model from `cmd/api/behavior/testmain_test.go`.

### Assertion Style

Prefer assertions that validate observable outcomes:

- status code
- response body
- persistence effects
- cross-resource constraints

Avoid tests that only prove implementation details or overfit to a single fixture value.

Never rely on exact list counts when seeded data may also be present.

## Workflow

### 1. Understand The Request

Identify:

- the domain (`Todo`, `User`, or both)
- the action (`Create`, `List`, `Get`, `Update`, `Delete`, filter behavior, validation path, and so on)
- the expected observable outcome
- whether the user is describing new behavior, changed behavior, or missing coverage

### 2. Inspect Current Coverage

Check whether the behavior already exists in:

- `cmd/api/behavior/todo_test.go`
- `cmd/api/behavior/user_test.go`
- relevant unit tests under `internal/...`

Avoid duplicating an existing behavior test unless the request is to update or replace it.

### 3. Inspect The Implementation

Read the relevant handlers, routes, and related persistence code so the requested test matches the real API contract.

If the request references a recent code change, inspect the relevant diff or changed files before proposing behavior coverage.

### 4. Decide The Correct Test Surface

Choose one of these outcomes:

- behavior test only
- unit test only
- both behavior and unit tests

If the request is really about DB failures or mocked branches, say that it belongs in unit tests instead of behavior tests.

### 5. Use The Approved Behavior-Test Path

If behavior test case changes are needed:

- use the `/behavior-test` workflow
- do not directly edit `cmd/api/behavior/todo_test.go`
- do not directly edit `cmd/api/behavior/user_test.go`

If `/behavior-test` is unavailable, stop and tell the user that behavior test case files cannot be updated manually in this repo.

## Output Shape

Return a concise result with:

- the behavior to add, update, or confirm
- the correct test surface: behavior tests, unit tests, or both
- the target domain and expected behavior-test name when relevant
- whether `/behavior-test` was used or whether the task is blocked because it is unavailable
- any follow-up verification that should run after the behavior-test update

If blocked, say so directly and include the exact behavior that still needs coverage so the user can run the correct workflow next.

## Verification Guidance

After behavior-test changes are made through the approved workflow, expect to run:

- `task behavior:diff`
- `task test:behavior:coverage`

If the related production code also changed, run the broader repo checks that match the change, including:

- `task lint`
- `task test:unit:coverage`

## Rules

- Treat behavior tests as the source of truth for observable API behavior.
- Never directly edit `cmd/api/behavior/todo_test.go` or `cmd/api/behavior/user_test.go`.
- Do not modify shared behavior-test infrastructure unless it is necessary and explicitly in scope.
- Base requests and coverage suggestions on externally visible behavior, not guessed internals.
- Prefer one behavior per test name when preparing or reviewing requested coverage.
- Call out missing success, failure, validation, and not-found cases when they matter.
