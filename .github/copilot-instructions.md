# Copilot Instructions — Review Behavior, Not Code

These instructions are for GitHub Copilot PR reviews in this repository.

Start by consulting `AGENTS.md` for the repository's current workflow rules, verification requirements, behavior-test policy, and file-specific constraints. Treat `AGENTS.md` as the source of truth for repo mechanics. Do not invent alternative workflows when that file already defines one.

## Review Goal

This repository uses a behavior-driven trust harness so humans can review **behavior** instead of re-reviewing large volumes of implementation.

The intended review model is:

```text
Review behavior -> trust tests -> avoid implementation-heavy review unless behavior or tests look suspicious
```

Your review should help maintain this standard:

> If CI is green and the behavior looks correct, the PR should be safe to merge.

## What To Prioritize In PR Review

Order your attention like this:

1. Behavior changes
2. Test quality and trustworthiness
3. Missing behavior coverage or edge cases
4. Implementation risks only when behavior or tests suggest a problem

Do not default to line-by-line implementation commentary when the PR's behavior is clearly specified, behavior changes are intentional, and the tests are trustworthy.

## Behavior-First Review

Treat behavior tests as the clearest specification of user-visible behavior.

When reviewing a PR:

- look for new, modified, or removed behavior
- check whether the changed behavior appears intentional
- focus on status codes, validation responses, response bodies, persistence effects, and cross-resource behavior
- call out surprising defaults, silent behavior drift, and inconsistent API semantics

If behavior tests changed, review the behavior diff before worrying about implementation details.

If a PR adds or changes observable API behavior and there is no corresponding behavior-test addition or update, flag that as likely missing specification coverage.

Infer expected behavior-test coverage from the user-visible surface of the change, not only from the tests already present.

Strong signals that behavior-test coverage may need to change include:

- new endpoints
- new query parameters
- new success or error response modes
- new validation behavior
- changed not-found, conflict, or authorization behavior

When a PR changes observable API behavior without a corresponding behavior-test update, that should usually be the primary finding unless a more severe correctness issue exists.

Treat missing behavior-test coverage for a user-visible change as a missing specification problem, not just a generic testing gap.

## Test Trustworthiness

The trust harness only works if the tests are meaningful.

Flag tests that:

- only execute code paths without proving outcomes
- assert too little to establish real behavior
- hardcode values in ways that make overfitting easy
- cover only the happy path when the PR adds new conditions, failure paths, or response modes

Prefer tests that verify observable outcomes such as:

- status codes
- response bodies
- persistence effects
- cross-resource constraints

Be alert for implementations that appear to satisfy a narrow test case instead of the real rule.

## When To Inspect Implementation Closely

Inspect implementation details more closely when:

- behavior changed without clear specification
- tests look weak, incomplete, or gameable
- expected behavior-test coverage is missing for a user-visible change
- there are missing negative cases or edge cases
- the code appears shaped to satisfy fixtures rather than general rules
- the behavior diff suggests a regression or an unreviewed semantic change

In those cases, implementation review is a follow-up tool for explaining the risk, not the primary review mode.

Do not spend review effort on speculative implementation concerns unless you can connect them to a concrete behavior risk, regression risk, or test-trust issue.

## Critical Repo Guardrails

Keep these constraints in mind during review even if other context is noisy:

- follow `AGENTS.md` for repo-specific verification and testing expectations
- behavior test case files in `cmd/api/behavior/` may need to change when observable behavior changes, but those updates should go through the `/behavior-test` workflow rather than direct manual edits
- new observable API behavior should come with behavior-test coverage, and new internal branches or DB-error paths should come with unit coverage where appropriate

When reviewing a PR, do not infer from the git diff alone that changes to `cmd/api/behavior/` violated the `/behavior-test` workflow. Assume those test-case changes were produced through the approved workflow unless the PR explicitly says otherwise or the content of the tests suggests a real issue.

## Review Output

When producing review feedback:

- lead with findings, not summary
- for behavior-related PRs, prefer this order: missing or changed observable behavior, missing behavior-test/spec coverage, weak or gameable tests, then implementation risks tied to those issues
- prioritize behavior regressions, weak tests, missing cases, and suspicious behavior diffs
- recommend the missing test or behavior coverage when that is the main issue
- keep implementation-style nits secondary unless they create a real behavior or reliability risk

The point of this file is not to restate the entire repo handbook. It is to keep Copilot's PR review behavior aligned with the repository's trust model: humans review behavior, CI enforces constraints, and implementation detail only becomes the focus when the harness gives a reason to distrust the change.
