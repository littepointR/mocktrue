# MockTrue Agent Instructions

## Multi-Agent Development Workflow

For non-trivial feature work, bug fixes, refactors, or verification tasks, use a multi-agent development loop when the tool environment supports sub-agents.

Each sub-agent must have one clear responsibility. Do not assign overlapping ownership unless the later agent is explicitly doing review or verification.

Recommended roles:

- Requirements analysis agent: read-only. Confirm scope, assumptions, acceptance criteria, and known gaps.
- Development agent: write-enabled only for a bounded file/module set. Implement the smallest change that satisfies the agreed scope.
- Code review agent: read-only. Prioritize correctness, regressions, lifecycle/resource issues, maintainability, and missing tests.
- Test verification agent: read-only. Run targeted tests first, then broader tests/type checks/builds as appropriate, and report exact commands and results.

Workflow:

1. Start with explicit assumptions and success criteria.
2. Split work into single-responsibility agents with disjoint write scopes.
3. Keep analysis/review/test agents read-only.
4. Tell write-enabled agents they are not alone in the codebase, must not revert unrelated changes, and must respect existing dirty files.
5. Integrate returned findings in the main agent, then fix any blocking issues.
6. Run a second review/test pass after fixes when the change is substantial.
7. Report remaining gaps separately from completed work.

Verification expectations:

- For frontend changes, run relevant targeted Vitest tests, then `npm test -- --run`, `npx vue-tsc --noEmit`, and `npm run build:dev` when feasible.
- For backend changes, run the relevant Go tests and any module-specific checks.
- Mention any tests that were skipped or could not be run.

Do not use multi-agent delegation for trivial one-file edits, simple command output requests, or documentation-only changes unless explicitly requested.
