---
name: feature-quality
description: Plan → review → implement → review → fix workflow for new features
---

# Feature Quality Workflow

Follow this workflow strictly:

## Step 1 — Use existing plan
Assume the user already entered plan mode and a plan exists.
If no plan exists yet, create a concise implementation plan.

Do not implement anything yet.

---

## Step 2 — Review plan (subagent)

Spawn a subagent to review the plan.

Subagent instructions:

You are a senior engineer reviewing an implementation plan.

Your job is to find:
- Missing steps
- Risky design decisions
- Overengineering
- Unclear requirements
- Edge cases

Be concise. Focus only on meaningful issues.

Return:
- Issues found
- Suggested improvements

---

## Step 3 — Refine plan

Update the plan based on the review.

Keep the plan concise and implementation-focused.

---

## Step 4 — Implement

Implement the refined plan.

Work step-by-step.
Keep changes small and safe.

---

## Step 5 — Review code (subagent)

Spawn a subagent to review the implementation.

Subagent instructions:

You are reviewing code changes.

Focus on:
- Bugs
- Edge cases
- Error handling
- Readability
- Consistency with existing code

Be concise.

Return:
- Issues found
- Suggested fixes

---

## Step 6 — Fix

Apply the suggested fixes.

---

## Step 7 — Verify (quick pass)

Perform a quick final review:

- Does the implementation match the plan?
- Any obvious issues?
- Any missing steps?

Fix minor issues if found.

---

## Rules

- Keep output concise
- Do not overengineer
- Prefer small changes
- Stop after workflow completes
