# MockTrue Agent Instructions

## Multi-Agent Development Workflow

Use the multi-agent development loop as the default workflow for non-trivial feature work, bug fixes, refactors, verification tasks, and release-quality changes.

If the tool environment supports sub-agents, delegate each phase to focused agents. If sub-agents are unavailable, the main agent must still follow the same loop sequentially: requirements analysis, implementation, code review, and test verification.

Each sub-agent must have one clear responsibility. Do not assign overlapping ownership unless the later agent is explicitly doing review or verification.

Recommended roles:

- Requirements analysis agent: read-only. Confirm scope, assumptions, acceptance criteria, and known gaps.
- Development agent: write-enabled only for a bounded file/module set. Implement the smallest change that satisfies the agreed scope.
- Code review agent: read-only. Prioritize correctness, regressions, lifecycle/resource issues, maintainability, and missing tests.
- Test verification agent: read-only. Run targeted tests first, then broader tests/type checks/builds as appropriate, and report exact commands and results.

Workflow:

1. Start with explicit assumptions and success criteria.
2. Split work into single-responsibility phases or agents with disjoint write scopes.
3. Keep analysis/review/test phases read-only.
4. Tell write-enabled agents they are not alone in the codebase, must not revert unrelated changes, and must respect existing dirty files.
5. Integrate returned findings in the main agent, then fix any blocking issues.
6. Run a second review/test pass after fixes when the change is substantial.
7. Report remaining gaps separately from completed work.

Verification expectations:

- For frontend changes, run relevant targeted Vitest tests, then `npm test -- --run`, `npx vue-tsc --noEmit`, and `npm run build:dev` when feasible.
- For backend changes, run the relevant Go tests and any module-specific checks.
- Mention any tests that were skipped or could not be run.

Do not use multi-agent delegation for trivial one-file edits, simple command output requests, or documentation-only changes unless explicitly requested.

## Repository Collaboration Conventions

- Treat this repository as solo-maintained but automation-friendly: agents may implement, verify, review, push, and open PRs when requested; do not merge unless the user explicitly asks.
- Preserve unrelated dirty files. Always inspect `git status --short --branch` before editing or staging, and stage only files that belong to the current task.
- Do not commit local agent artifacts such as `.hermes/` plans, scratch worktrees, logs, or temporary state files unless the user explicitly asks for them.
- Prefer repository instructions such as this file over local Hermes/profile skills, because local skills may not exist on other machines.
- Avoid changing globally installed or update-managed agent skills from inside this repo. If a workflow rule is project-specific, document it here instead.

## Review and Kanban Gating Policy

- If a task graph already has downstream verify/review cards, implementation agents should complete with a rich handoff instead of blocking with `review-required` solely because code changed.
- When review is needed and no review card exists, create a separate read-only reviewer task/card that depends on the implementation task.
- Reviewer tasks should inspect the diff, scope, tests, generated files, secrets, and regression risks. They should either pass with evidence or create/block on specific follow-up fix scope.
- Use a human `review-required` block only for explicit user approval, product decisions, credentials/permissions, destructive operations, or ambiguity an agent cannot resolve safely.

## Verification Commands

- Backend/full Go checks: `make test`.
- Backend lint: `make lint` requires `golangci-lint` to be installed; if missing, report it as an environment blocker rather than claiming lint passed.
- Backend build: `make build`.
- Frontend uses `pnpm` from `frontend/`:
  - Targeted tests: `pnpm exec vitest run <test files>`.
  - Full frontend tests when feasible: `pnpm test -- --run`.
  - Typecheck: `pnpm exec vue-tsc --noEmit`.
  - Development build: `pnpm run build:dev`.
- For Wails binding-sensitive changes, regenerate/check bindings only when exported Go service/model signatures changed. Review generated diffs carefully and do not stage unrelated binding churn.
- `git diff --check` should pass before committing.

## GUI Testing Strategy

- Prefer fast, stable automated coverage over broad real-window clicking: Go unit/integration tests, Vitest store/component tests, TypeScript typecheck, build, and contract/binding drift checks are the default PR gate.
- For Vue/Wails UI behavior, test stores and components with mocked backend/Wails bindings first.
- Add browser/dev-server or real desktop GUI smoke tests only for critical end-to-end flows; keep them small because they are slower and more fragile.
