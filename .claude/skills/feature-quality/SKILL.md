---
name: feature-quality
description: Use after Codex or Claude has already created an implementation plan and that plan is available in context. Review and refine the plan first, then execute it through tests, implementation, review, and verification. Do not use for initial planning, pure analysis, or tiny fast patches.
---

# Feature Quality

Use this skill after a planning workflow has already produced a plan and that plan is available in context.

## Goal

Complete the requested feature or refactor with a deliberate sequence:

1. review the existing plan
2. improve the plan if needed
3. write or update tests from the refined plan
4. implement incrementally
5. review the result critically
6. verify with the narrowest checks that match the change

The plan is the starting artifact for this skill. Do not treat this as a general-purpose planning skill.

If another skill is needed during the task, use it and then resume this workflow from the next unfinished step.

When the runtime supports user-facing progress updates, show the workflow as a short checklist and update it as phases complete.

## Workflow State

Track the current phase and resume from the first incomplete phase:

- Phase 1 - Review The Plan
- Phase 2 - Improve The Plan
- Phase 3 - Tests First
- Phase 4 - Implement
- Phase 5 - Review
- Phase 6 - Verify

Always resume from the first unfinished phase after interruptions, nested skill use, or tool-heavy branches.

## Execution Rules

- Continue executing phases until all phases are complete. Do not stop early unless blocked.
- Prefer incremental, small-scope changes. Avoid large rewrites unless the user explicitly wants them or the plan requires them.
- Prefer the narrowest necessary execution in each phase. Avoid unnecessary broad reads, reviews, test runs, or refactors.
- When tests are added in this workflow, let them define the intended behavior that implementation must satisfy.
- If any phase reveals that a major assumption was wrong, pause, update or refine the plan, and resume from the appropriate phase instead of pushing forward on stale assumptions.
- **When invoking a nested skill, always use the Agent tool — never the Skill tool.** The Skill tool injects the skill's SKILL.md as a new Human message, creating a new conversational turn that ends the current response and breaks feature-quality continuity. The Agent tool returns a result within the current response, keeping the workflow uninterrupted. Brief the agent with the skill's SKILL.md content, relevant file context, and the specific task. After the agent returns, restate the current phase and continue immediately. If the agent stalls or fails to complete its work, re-invoke it — do not substitute direct edits for a skill's output.

## Required Starting Point

Before doing any implementation work:

- If plan mode is active, call ExitPlanMode first — this skill executes code and cannot run under plan mode.
- assume a plan already exists because a planning workflow just ran
- read that plan from context before touching code
- treat the plan as required input, not an optional hint

If the plan is unexpectedly missing or too incomplete to execute safely, stop and say that the task needs a usable plan before this skill can continue.

After reading the plan, state briefly:

- what will change
- what tests will be written or updated
- what verification will run

If user-facing progress updates are available, present that summary as a short checklist of remaining phases.

Then continue with the workflow below.

## Read Order

Read only what you need.

1. Read the plan in context first.
2. Read the relevant implementation files to understand the current code the plan refers to.
3. Read the existing tests in the affected area before adding or changing test coverage.
4. Read neighboring files only if integration points or risks are still unclear.

## Phase 1 — Review The Plan

Spawn an Explore subagent to review the plan independently. Do not review it yourself — the subagent's independence is the point; the same model that shaped the plan will tend to defend it.

Before spawning, read the key implementation files the plan references so you can include their current content in the subagent prompt.

The subagent prompt must include:
- The full plan text
- The current content of the files the plan will change
- This brief: "Challenge this plan. Find: incorrect assumptions, test scenarios that won't work as described, missing edge cases, validation gaps, and risky design decisions. Do not suggest style changes. Return only actionable issues with brief explanations."

Wait for the subagent to return, then carry its findings into Phase 2.

## Phase 2 — Improve The Plan

Refine the plan based on the review.

- keep it incremental
- keep the scope aligned with the user request
- add missing test or verification work
- remove unnecessary complexity

If the plan changes materially, state the important adjustments before continuing. If the review does not change the plan, say that the plan still stands.

## Phase 3 — Tests First

**Write all tests before touching any implementation file.** This is a hard constraint, not a preference. All test types applicable to the change must be written in this phase — do not defer any test type to after implementation.

When the change introduces or alters observable behavior:
- Write the tests that would fail against the current code.
- Cover the main success path and the important failure or edge paths.
- Follow the existing test structure and assertion style, including all test types the repo uses for this kind of change.
- Finish all test updates before editing implementation files.

After writing tests, review their intent:
- Do they prove the real outcome, or just that the code ran?
- Do any assertions assume output format, casing, encoding, or structure that the implementation has not yet been written to produce? Confirm these assumptions are correct before moving on.
- Are there brittle assumptions or implementation-detail overfitting?
- Is coverage missing for any requested behavior?

Fix test intent problems before moving on.

If the change truly does not alter any exercised behavior, state that explicitly and skip this phase.

## Phase 4 — Implement

Only open implementation files after tests are written.

- Make changes in small steps, one logical unit at a time.
- Follow established patterns in the codebase.
- Handle boundaries and errors explicitly.
- Keep unrelated changes out of scope.

If implementation reveals a mismatch with the plan or a risky surprise, return to Phase 1 or Phase 2 as needed, then continue from the first unfinished phase.

## Phase 5 — Review

**Run static analysis first.** Before invoking `/simplify`, run the linter and fix any findings. If a `.semgrep/` directory or `.semgrep.yml` exists at the project root, also run `semgrep scan --config <config>` and fix those findings too. This ensures the review agents see already-compliant code — otherwise a finding caught later in Phase 6 requires a post-review fix that the agents never saw.

Then, before invoking `/simplify`, read the full body of every function that was changed or added — not just the diff. Review agents that only see the diff can make incorrect suggestions when the relevant context lives outside the changed lines (e.g. a variable whose meaning is determined by surrounding state they cannot see).

Invoke the `/simplify` skill using the Skill tool, passing the changed file names and the full bodies of changed functions as context in the args.

**Apply findings critically.** Before applying any suggested change, verify it is correct given the full function body and surrounding state. If a finding would alter shared state or a variable used outside the diff, check all callers. Note false positives explicitly and skip them — do not apply a fix just because an agent suggested it.

If the Skill tool returns an error indicating `/simplify` is unavailable in the current runtime, document that explicitly, then perform a self-review covering:
- correctness bugs
- missing edge cases
- accidental behavior changes
- readability or naming problems
- unfinished branches or placeholder notes

Fix anything the review finds before moving to verification.

## Phase 6 — Verify

Follow the **Verification** section in `AGENTS.md` (or equivalent project configuration file). That section defines which commands to run and the conditions under which each applies.

If no `AGENTS.md` exists or it has no Verification section, run at minimum: a linter and the test suite most relevant to the changed code.

Report any verification gap explicitly if a required check could not run.

## Output Shape

Return a concise summary that includes:

- whether the plan was kept as-is or refined
- what changed
- any assumptions made
- whether tests were added, updated, or intentionally left unchanged
- what review was performed
- what verification ran
- any remaining risks, follow-ups, or blockers

## Completion Checks

Before finishing, confirm:

- the implementation matches the requested outcome
- tests cover new or changed behavior when appropriate
- unrelated changes were not pulled in accidentally
- the linter was run and clean before `/simplify` was invoked
- the `/simplify` skill was invoked before verification, or its unavailability was explicitly documented
- required verification was run or any gap was reported clearly
- no unfinished code paths or placeholder notes remain

## Guardrails

- Do not create a fresh plan here unless the existing one is missing or unusable, in which case stop instead of improvising a full replacement.
- Do not force test creation when the change does not warrant it.
- Do not expand scope just because a larger refactor would be nicer.
- Do not skip repository-specific review or verification requirements.
- Do not stop the workflow after implementation without review and verification.
- Do not stop the workflow after a nested skill completes. A nested skill output is a sub-task result — feature-quality ends only when all six phases are done.
