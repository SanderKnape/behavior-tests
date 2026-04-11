# Copilot Instructions — Behavior-Driven Trust Harness

This repository is a Go API backed by PostgreSQL. The safety model is simple:

- Do not trust implementation details on their own.
- Trust observable behavior, verified by the test harness.
- Treat behavior tests as the strongest expression of intended API behavior.

Your job is to help keep generated code, reviews, and test changes aligned with that model.

## Repository Facts To Anchor On

- Unit tests live under `internal/...`.
- Behavior tests live under `cmd/api/behavior/...`.
- Behavior tests act as specifications for user-visible API behavior.
- Test infrastructure for behavior tests lives in `cmd/api/behavior/testmain_test.go`.
- Schema migrations live in `internal/platform/db/migrations/`.
- Seeds live in `internal/platform/db/seeds/`.
- Tooling is managed with `mise`; task execution is managed with `Taskfile.yml`.

## Core Operating Rules

When generating or reviewing code:

1. Start from behavior, not implementation.
2. Preserve the intent of existing behavior tests unless the requested change is explicitly behavioral.
3. Prefer simple code that satisfies the specified behavior over clever abstractions.
4. Look for signs that code or tests are narrowly shaped to satisfy one case instead of the real requirement.

## Testing Model

### Unit Tests

Use unit tests to validate internal logic, error handling, and cases that benefit from mocking.

Prefer adding unit tests in the matching package, such as:

- `internal/todos/handler_test.go`
- `internal/users/handler_test.go`

Add unit tests for:

- DB error paths
- branching business logic
- validation paths that are easier to isolate with mocks

### Behavior Tests

Behavior tests verify observable API behavior end to end.

Naming convention:

```go
TestBehavior_<Domain>_<Action>_<Expectation>
```

Examples:

```go
TestBehavior_Todo_List_ReturnsSeededTodos
TestBehavior_Todo_Create_PersistsAndReturns
TestBehavior_Todo_Get_Returns404ForUnknownID
```

Behavior tests should describe outcomes a user or client can observe, such as:

- response status
- response body
- persistence effects
- validation failures

## Hard Rule: Do Not Edit Behavior Test Case Files Directly

Do not directly modify behavior test case files in `cmd/api/behavior/`, including:

- `cmd/api/behavior/todo_test.go`
- `cmd/api/behavior/user_test.go`

Those files must be updated through the `/behavior-test` skill so naming and structure stay consistent.

If a requested change requires behavior-test case updates and that skill is unavailable, stop and say so instead of editing the files manually.

You may modify shared behavior-test infrastructure in `cmd/api/behavior/testmain_test.go` when needed.

## When Adding Or Changing Functionality

Use this order of operations:

1. Clarify the intended observable behavior.
2. Add or update tests for that behavior using the approved workflow.
3. Add unit tests for internal branches and DB-error paths.
4. Implement the production change.
5. Run the narrowest verification that matches the change.

When introducing new code paths, do not rely on coverage gates to reveal missing tests. Add the tests in the same change before running coverage checks.

## Review Checklist

When reviewing code or AI-generated changes, check these areas in order.

### 1. Behavior Correctness

Ask:

- Does the change preserve existing specified behavior?
- If behavior changed, is the new behavior clearly intentional?
- Do status codes, validation responses, defaults, and persistence behavior make sense?

Examples of suspicious behavior changes:

- a missing validation error
- a `404` becoming `200`
- silent fallback defaults that were not requested
- updates that do not persist or return stale data

### 2. Test Quality

Prefer tests that verify outcomes, not implementation details.

Good behavior-oriented assertions include:

- response body validation
- database state validation
- end-to-end effects across request and persistence layers

Flag tests that:

- only assert status codes
- hardcode values in ways that make cheating easy
- fail to verify persistence or returned payloads
- only cover the happy path when error branches were added

Prefer assertions tied to the request input when appropriate:

```go
assert.Equal(t, input.Title, todo.Title)
```

Be cautious when a test overfits to a literal value without proving the real rule:

```go
assert.Equal(t, "Buy milk", todo.Title)
```

### 3. Test Gaming And Overfitting

Reject or question implementations that appear shaped only to satisfy the visible test cases.

Examples:

- hardcoded return values
- branch logic keyed to a fixture value rather than the rule
- skipping validation or persistence while still producing the expected response for one case

Suspicious example:

```go
return "Buy milk"
```

### 4. Coverage Intent

When functionality changes in an observable way, expect both:

- unit coverage for internal logic and failure branches
- behavior coverage for externally visible API behavior

Common missing cases:

- invalid input
- not found
- unauthorized or forbidden access
- partial update semantics
- DB failure handling

### 5. Behavior Diff Review

If behavior tests changed, inspect the behavior diff with `task behavior:diff`.

Focus on:

- new behavior
- modified behavior
- removed behavior

Confirm the diff matches the requested product change and does not hide regressions.

## Verification Expectations

Run the narrowest checks that match the change and say what you could not run.

Default for any code change:

- `task lint`
- `task test:unit:coverage`

Also run `task test:behavior:coverage` if you changed:

- API handlers
- database code
- migrations
- seeds
- behavior-test infrastructure

Also run `task build` if you changed:

- build wiring
- CLI startup
- dependencies

If behavior tests changed through the approved skill, also run:

- `task behavior:diff`
- `task test:behavior:coverage`

## Code Generation Expectations

Generated code should:

- match existing project patterns
- stay readable and direct
- avoid unnecessary abstraction
- implement the requested behavior fully, not minimally
- align with the tests' intent rather than exploiting their gaps

## Review Output Preference

When asked to review, prioritize findings over summary.

Order review feedback by:

1. behavior correctness
2. test quality
3. missing edge cases
4. implementation quality

If tests are missing, say which ones are needed:

- success case
- failure case
- edge case
- authorization case

## Final Goal

The goal is to keep this statement true:

> If CI is green and the behavior looks correct, the change is safe to merge.

Optimize your suggestions, code generation, and review comments for that standard.
