---
slug: qt-rust-skeleton-completion
status: available-tool-review-passed-native-momus-unavailable
intent: clear
review_required: true
pending-action: start execution with `$omo:start-work .omo/plans/qt-rust-skeleton-completion.md` if the native-Momus caveat is acceptable; rerun native Momus if multi_agent_v1 tools become available
approach: Finish the current Qt/Rust skeleton branch to the documented skeleton exit gates before any product feature migration. Keep scope limited to desktop/qt-rust, narrow environment/preflight fixes, matrix/QML guardrails, docs evidence, and CI after local build proof.
---

# Draft: qt-rust-skeleton-completion

## Components (topology ledger)
1. env-preflight | Current shell can prove or clearly block Qt/Rust prerequisites without global installs | active | `scripts/check-qt-rust-env.ps1`; T1/T2 evidence
2. rust-core-smoke | Rust staticlib smoke API builds and tests independently | active | `desktop/qt-rust/rust/src/lib.rs`; T4 evidence
3. qt-qml-build | CMake/Ninja builds the Qt executable, copies QML, and exposes CTest/app smoke gates | active | `desktop/qt-rust/CMakeLists.txt`; T5/T6 evidence
4. desktop-surface-smoke | Running app proves visible QML, Rust-originated smoke value, and lifecycle logging | active | `desktop/qt-rust/qml/shell/*.qml`; T7 evidence
5. migration-gates-docs | Readiness/gap/startup docs reflect actual skeleton evidence and remaining blockers | active | `docs/development/qt-rust-skeleton-startup.md`; `docs/development/migration-gap-report.md`; T9 evidence
6. ci-followup | New-stack CI coverage is added only after local build command exists and passes | active | `.github/workflows/ci.yml`; T8 evidence

## Open assumptions (announced defaults)
1. toolchain handling | Do not install or change global Qt/Rust/MSVC tools during execution; use current shell or `PORTWEAVE_QT_RUST_ENV_SCRIPT`, and if missing, stop with an explicit blocker | Project docs require a private machine-local bootstrap and shared docs must not hardcode local paths | reversible
2. branch base | Continue from current `feat/qt-rust-merged` branch rather than recreating the merge | It already contains `origin/master`, `origin/feat/qt-rust-skeleton`, and the smoke-action semantic fix | reversible
3. product scope | Treat toolbar file actions as `shell.smoke.*` only; do not mark `workspace.*` rows implemented | Skeleton exit gates precede first-wave product migration | reversible
4. Rust bridge | Keep the first-stage C ABI bridge; do not switch to `cxx-qt`, IPC, or pure Qt/C++ unless skeleton evidence triggers the ADR stop condition | ADR-0006 makes the C ABI bridge the skeleton decision gate | reversible
5. QA evidence | Require both command evidence and agent-driven desktop surface evidence; tests alone are insufficient | Skeleton exit gates require launch, visible QML content, Rust smoke visibility, and lifecycle behavior | reversible
6. monitor fallback | If only one monitor exists, run resize/minimize/restore and document the monitor-count fallback instead of blocking forever | Cross-screen movement is hardware-dependent; lifecycle proof still has an agent-executable fallback | reversible

## Findings (cited - path:lines)
- Current branch is `feat/qt-rust-merged`, ahead of `origin/master` by three commits. Dirty worktree has pre-existing `go.mod`, `go.sum`, `.hermes/`, and `third_party/` out of scope.
- Current shell has `node`, `python`, `make`, `cmake`, and `ninja`; it does not expose `cargo`, `rustc`, `qmake`, or `cl.exe`, and `QT_ROOT`, `CMAKE_PREFIX_PATH`, and `PORTWEAVE_QT_RUST_ENV_SCRIPT` are empty.
- `desktop/qt-rust/**` already exists and contains the skeleton; the plan must complete/verify it rather than recreate it.
- `desktop/qt-rust/CMakeLists.txt` wires Qt6 Core/Gui/Qml/Quick, Cargo staticlib build, executable `PortWeaveQtRust`, and QML copy beside the executable.
- `desktop/qt-rust/rust/src/lib.rs` exports C ABI smoke functions.
- `desktop/qt-rust/qml/shell/EditorGroups.qml` displays `Rust smoke result: ` plus `appBridge.smokeMessage` and lifecycle event log.
- `docs/development/qt-rust-skeleton-startup.md` defines skeleton exit gates: preflight, migration matrix, CMake configure/build, app launch, visible QML, Rust smoke value visible in QML, cross-screen/resize/minimize/restore behavior, lifecycle/DPI/screen observable events, and `git diff --check`.
- `docs/development/migration-readiness.md` blocks first product feature implementation until the skeleton launches, Rust event reaches QML, Windows lifecycle smoke passes, and first-wave rows remain `contracted`.
- `docs/development/migration-gap-report.md` has stale evidence about skeleton file absence/toolchain state and says the QML action featureId check should be extended after QML skeleton files exist.
- `.github/workflows/ci.yml` currently has existing repo CI jobs but no Qt/Rust skeleton coverage; CI should be added after local build proof.

## Decisions (with rationale)
- Plan intent: CLEAR. The desired outcome is known from the docs: finish the Qt/Rust skeleton exit gates, not redesign the migration.
- Tier: HEAVY. The plan crosses new Qt/C++/QML/Rust build layers, desktop GUI QA, CI/docs, and architecture decision gates.
- Test strategy: TDD for behavior/checker changes; tests-after is acceptable for narrow docs/evidence updates; every todo includes command or desktop-surface QA evidence.
- Approval: user replied `批准`; approval authorizes writing the plan only, not implementation.
- High-accuracy review: requested by user after plan delivery. Run dual review before handoff and record receipts here.

## Scope IN
- Preserve and continue current `feat/qt-rust-merged` branch.
- Bring the existing Qt/Rust skeleton to the documented exit gates under `desktop/qt-rust`.
- Fix only narrow skeleton/build/preflight issues needed for the skeleton to build and launch.
- Add/update tests/smoke commands for Rust C ABI, Qt/C++ bridge/lifecycle seams, QML boundaries, and launch/lifecycle evidence.
- Extend matrix/QML featureId checks now that QML skeleton files exist.
- Update migration startup/gap/readiness docs only with concrete evidence and command results.
- Add CI for migration preflight and Qt/Rust configure/build only after the local build command exists and passes.

## Scope OUT (Must NOT have)
- No product feature migration beyond skeleton smoke.
- No `workspace.*`, serial, graph, protocol, MCP, installer, or packaging implementation.
- No QML direct Rust calls.
- No Rust dependency on Qt/QML types.
- No rewrite or removal of existing Go/Wails/Vue code.
- No staging or committing `.hermes/`, scratch logs, `third_party/`, or unrelated `go.mod/go.sum` changes.
- No hardcoded local machine paths in shared docs.
- No global toolchain install/change unless explicitly approved later.

## Open questions
- None blocking for plan writing. Adopted defaults are encoded in the plan.

## Approval gate
status: approved
pending action: `$omo:start-work .omo/plans/qt-rust-skeleton-completion.md` if the available-tool review is sufficient; rerun native Momus first if strict native subagent compliance is required

## High-accuracy review receipts
- native Momus pass 1: ITERATE. Blocking issues: T7 dependency mismatch, T7 non-exact GUI QA command, F1-F4 not executable enough.
- native Momus pass 2: ITERATE. T7 fixed; remaining blockers: F1 inline PowerShell not reliably executable, F4 did not write declared summary file and did not handle `rg` no-match pass semantics.
- native Momus pass 3: OKAY. Prior blockers resolved in submitted plan/draft.
- independent Codex CLI earlier pass: ITERATE/blockers. Required explicit todo for final helper scripts; exact negative commands for T3/T5/T8; explicit `rg` no-match handling for T4/T6/T9.
- independent Codex CLI pass 1: failed before review due isolated CODEX_HOME missing auth, 401 Unauthorized.
- independent Codex CLI pass 2: timed out after auth copy; result pending or unusable until output appears.
- Momus prompted/profile CLI pass: OKAY. Review directory `.omo/tmp/momus-prompted-readonly-review-20260702-182909`; session `019f2260-12d6-7c22-bf2f-3cf7f682f89f`; command used minimal `CODEX_HOME`, read-only sandbox, gpt-5.5/xhigh, and embedded Momus role instructions. Verdict: plan is executable; references covered, dependencies coherent, QA evidence concrete, and F1-F4 startable after planned helper scripts. Note: this is not the native `multi_agent_v1.spawn_agent` Momus subagent path because that tool is not exposed in this session.
- independent Codex CLI embedded final pass: OKAY. Review directory `.omo/tmp/independent-codex-embedded-review-20260702-190700`; session `019f2283-5019-74f2-9dfd-9ff2c2e3718c`; command used the same minimal `CODEX_HOME` as the Momus profile review, read-only sandbox, gpt-5.5/xhigh, and embedded plan/draft/reference-existence text with no shell/file reads requested. Verdict: references are accounted for or planned as new helper scripts, todo dependencies align, final helpers and F1-F4 are startable after T10/T7, and prior blockers are addressed.
- Momus availability check: strict native `multi_agent_v1.spawn_agent` / `wait_agent` / `close_agent` tools were not exposed in this session. `C:\Users\pc\.codex\agents\momus.toml` exists and `codex exec -p momus` profile smoke previously returned `MOMUS_PROFILE_OK`, so Momus is usable through the profile/CLI path but not through the native subagent path here.
- fix/retry summary: patched T7 dependency to T6, made `desktop/qt-rust/scripts/smoke-gui.ps1` required with exact happy/failure commands, replaced brittle F1/F4 inline PowerShell with required repo-local final-check scripts, moved T10 into Wave 1 so `assert-no-rg-match.ps1`, `final-plan-compliance.ps1`, and `final-scope-fidelity.ps1` exist before any todo invokes them, added exact smoke/failure commands for all three helpers, replaced T3/T5/T8 failure scenarios with exact commands, routed T4/T6/T8/T9 forbidden-match scans through explicit no-match helper semantics, aligned T4/T5/T6/T10/T11 dependencies, made F2 propagate `git diff --check` failures, and reran the available profile/CLI review path to OKAY. Strict native Momus subagent receipt remains unavailable because the tool is not exposed in this session.
