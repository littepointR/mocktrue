# qt-rust-skeleton-completion - Work Plan

## TL;DR (For humans)
**What you'll get:** A finished Qt/Rust desktop skeleton branch that proves the new stack can build, launch, show QML, show a Rust-originated smoke value, and survive the first Windows lifecycle checks before any product migration starts.

**Why this approach:** The existing branch already contains the skeleton, so the plan completes and verifies it instead of recreating it. Missing local Qt/Rust/MSVC tools are treated as a prerequisite blocker unless a private bootstrap script restores them.

**What it will NOT do:** It will not migrate workspace, serial, graph, protocol, MCP, installer, packaging, or Wails behavior. It will not let QML call Rust directly, make Rust depend on Qt/QML, install global toolchains, or commit unrelated dirty files.

**Effort:** Large
**Risk:** High - success depends on restoring the Windows Qt/Rust/MSVC toolchain and proving real desktop GUI behavior, not just passing text checks.
**Decisions to sanity-check:** The skeleton stays limited to `shell.smoke.*` and module-navigation controls; CI is added only after local configure/build passes; single-monitor hosts use a documented fallback instead of blocking forever on cross-screen proof.

Current gate: available-tool review passed. Momus profile/CLI review is OKAY and independent Codex CLI review is OKAY; strict native `multi_agent_v1` Momus subagent tools were not exposed in this session. Start execution with `$omo:start-work .omo/plans/qt-rust-skeleton-completion.md` if that caveat is acceptable, or rerun native Momus first if the tool becomes available. Full execution detail follows below.

---

> TL;DR (machine): Large/high-risk skeleton completion plan; finish environment gates, matrix/QML fences, Rust/Qt build tests, desktop smoke evidence, docs, CI, and cleanup without product migration.

## Scope
### Must have
- Continue from the current `feat/qt-rust-merged` branch and preserve the existing merge/smoke-action commits.
- Treat the already-created `desktop/qt-rust/**` skeleton as the starting point; complete and verify it rather than recreating it.
- Keep edits limited to `desktop/qt-rust/**`, narrow `scripts/check-qt-rust-env.ps1`, `scripts/check-migration-matrix.py` plus its tests/fixtures, `.github/workflows/ci.yml` after local build passes, and documentation evidence under `docs/development/qt-rust-skeleton-startup.md` and `docs/development/migration-gap-report.md`.
- Prove the documented skeleton exit gates: preflight, migration matrix, Rust tests, CMake configure/build, executable launch, visible QML, Rust-originated smoke result visible in QML, lifecycle/DPI/screen observability, resize/minimize/restore behavior, and `git diff --check`.
- Extend the QML action/feature gate now that skeleton QML exists, while allowing only skeleton smoke actions and pure module navigation.
- Add required repo-local PowerShell verification helpers under `desktop/qt-rust/scripts/` for GUI smoke, no-match `rg` assertions, final plan compliance, and final scope fidelity.
- Record command and desktop QA evidence under `.omo/evidence/qt-rust-skeleton-completion/` during execution, but do not commit `.omo/evidence` unless the user explicitly asks.
- Stage only files belonging to the current task; preserve unrelated `go.mod`, `go.sum`, `.hermes/`, `third_party/`, and unrelated `.omo` artifacts.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- No product feature migration beyond skeleton smoke controls.
- No implementation or acceptance of `workspace.*`, serial, graph, protocol, virtual backend, MCP, installer, updater, Modbus, FECbus, or packaging behavior.
- No conversion of `shell.smoke.new/open/save/save_as` into real workspace actions.
- No QML direct Rust calls, FFI loading, process spawning, or parsing backend error strings.
- No Rust dependency on Qt, QML, QObject, `qmetaobject`, `cxx-qt`, or generated QML types.
- No rewrite, removal, or behavioral change to the existing Go/Wails/Vue product path.
- No hardcoded local checkout paths, install roots, usernames, or personal bootstrap scripts in shared docs.
- No global Qt/Rust/MSVC installation or machine-wide configuration changes unless the user later explicitly authorizes them.
- No CI Qt/Rust build job until the local configure/build command passes and the required toolchain setup is expressed through portable environment variables.

## Verification strategy
> Zero human intervention - all verification is agent-executed. Desktop GUI evidence is agent-driven and recorded; any physical hardware limitation is recorded as a residual risk, not hidden.
- Test decision: TDD for behavior/checker changes; tests-after for narrow docs and evidence updates; no product-feature tests because product features stay out of scope.
- Evidence root: `.omo/evidence/qt-rust-skeleton-completion/`.
- Baseline checks:
  - `git status --short --branch`
  - `git log --oneline --decorate -5`
  - `node --version`, `python --version`, `make --version`, `cmake --version`, `ninja --version`
  - `powershell -NoProfile -Command "Get-Command cargo,rustc,qmake,cl.exe -ErrorAction SilentlyContinue | Select-Object Name,Source"`
- Environment negative check:
  - `powershell -NoProfile -Command "Remove-Item Env:QT_ROOT,Env:CMAKE_PREFIX_PATH,Env:PORTWEAVE_QT_RUST_ENV_SCRIPT -ErrorAction SilentlyContinue; & .\scripts\check-qt-rust-env.ps1"`
  - Expected: nonzero, friendly diagnostic that names the missing prerequisite, no PowerShell null-argument stack trace.
- Environment positive check after bootstrap:
  - If `PORTWEAVE_QT_RUST_ENV_SCRIPT` is set, load it in the same shell and run `powershell -ExecutionPolicy Bypass -File scripts\check-qt-rust-env.ps1`.
  - Expected: `Qt/Rust environment preflight OK`.
  - If no bootstrap or toolchain is available, stop execution after recording the blocker; do not fake build/run success.
- Matrix/QML checks:
  - `python scripts\check-migration-matrix.py`
  - Add a failing fixture/temp-copy case that contains an actionable QML control without an allowed `featureId`/action ID and assert the checker fails.
  - Add a passing fixture/temp-copy case for `shell.smoke.*` actions and `serial`/`settings` module navigation.
- Rust checks:
  - `cargo test --manifest-path desktop\qt-rust\rust\Cargo.toml`
  - Boundary helper: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "Qt|QML|QObject|qmetaobject|cxx_qt" -Path desktop/qt-rust/rust/src,desktop/qt-rust/rust/Cargo.toml -OutFile .omo\evidence\qt-rust-skeleton-completion\t4-rust-boundary.txt` should treat no Rust Qt coupling as pass, forbidden matches as failure, and `rg` errors as tool errors.
- Qt/CMake checks:
  - `cmake -S desktop\qt-rust -B build\qt-rust -G Ninja -DCMAKE_BUILD_TYPE=Debug`
  - `cmake --build build\qt-rust`
  - `ctest --test-dir build\qt-rust --output-on-failure` after CTest is added.
  - `.\build\qt-rust\PortWeaveQtRust.exe --smoke-exit`
- QML/Rust boundary checks:
  - `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "portweave_rust_|extern|Library|ffi|Process" -Path desktop/qt-rust/qml -OutFile .omo\evidence\qt-rust-skeleton-completion\t6-qml-boundary.txt` should treat no matches as pass, forbidden matches as failure, and `rg` errors as tool errors.
- Desktop GUI smoke:
  - Launch `.\build\qt-rust\PortWeaveQtRust.exe` via PowerShell.
  - Capture initial, resized, minimized/restored, and final restored screenshots.
  - Verify the visible window title is `PortWeave Qt/Rust Skeleton`.
  - Verify the UI visibly contains the Rust smoke result and lifecycle/status text.
  - Drive resize, minimize, restore, and screen move when more than one monitor is available; on one-monitor hosts, record monitor count and run the resize/minimize/restore fallback.
  - Stop the process and record PID, exit/cleanup result, screenshots, and final observed lifecycle event.
- Cleanup:
  - `git diff --check`
  - `git status --short --branch`
  - Path-leak scan over touched shared docs/scripts for `D:\`, `C:\Users\`, and personal bootstrap paths.

## Execution strategy
### Parallel execution waves
- Wave 0: Baseline and blocker handling. Verify branch, dirty worktree, and local toolchain facts before touching product files.
- Wave 1: Guardrail/checker hardening. Fix environment diagnostic quality, add shared verification helpers, and add QML action/feature gate tests before modifying the checker.
- Wave 2: Core skeleton verification. Add/adjust Rust, Qt/C++, CTest, QML lint, and app smoke harnesses around the existing skeleton.
- Wave 3: Desktop surface proof. Run the real Windows GUI smoke and capture evidence once build/run gates pass.
- Wave 4: Docs and CI. Update stale documentation evidence and add CI only after local build proof exists.
- Wave 5: Cleanup and final review. Run final checks, confirm scope fidelity, and prepare a narrow commit set.

### Dependency matrix
| Todo | Depends on | Blocks | Can parallelize with |
| --- | --- | --- | --- |
| T1 | none | T2, T3, T4, T5, T6, T7, T8, T9, T10, T11 | none |
| T2 | T1 | T3, T4, T5, T6, T8, T9, T10 | none |
| T3 | T1, T10 | T8, T9 | T4 after T10 |
| T4 | T2, T10 | T5, T8, T9 | T3 |
| T5 | T2, T3, T4 | T6, T8, T9 | none |
| T6 | T5, T10 | T7, T8, T9 | none |
| T7 | T6 | T9, T11 | none |
| T8 | T3, T4, T5, T6, T10 | T9, T11 | none |
| T9 | T6, T7, T8, T10 | T11, final verification | none |
| T10 | T2 | T3, T4, T6, T8, T9, T11, final verification | none |
| T11 | T9, T10 | final verification | none |

## Todos
> Implementation + Test = ONE todo. Never separate.
<!-- APPEND TASK BATCHES BELOW THIS LINE WITH edit/apply_patch - never rewrite the headers above. -->
- [x] T1. Lock current branch state and toolchain prerequisite behavior
  What to do / Must NOT do: Capture the current branch, dirty worktree, and tool availability as execution facts. Treat missing `cargo`, `rustc`, `qmake`, `cl.exe`, `QT_ROOT`, `CMAKE_PREFIX_PATH`, or `PORTWEAVE_QT_RUST_ENV_SCRIPT` as a prerequisite blocker unless a private bootstrap script is already present. Do not install global tools or edit global configuration.
  Parallelization: Wave 0 | Blocked by: none | Blocks: T2, T3, T4, T5, T6, T7, T8, T9, T10, T11
  References (executor has NO interview context - be exhaustive): `.omo/drafts/qt-rust-skeleton-completion.md`; `docs/development/qt-rust-skeleton-startup.md`; `docs/development/migration-gap-report.md`; `scripts/check-qt-rust-env.ps1`; current dirty paths `go.mod`, `go.sum`, `.hermes/`, `third_party/`
  Acceptance criteria (agent-executable): Evidence file records `git status --short --branch`, latest commits, tool command availability, relevant Qt/Rust/MSVC env vars, and a go/no-go decision. If tools are missing and no bootstrap is set, executor stops after T2's friendly diagnostic fix and reports an explicit blocker instead of attempting build/run todos.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -NoProfile -Command "Get-Command node,python,make,cmake,ninja,cargo,rustc,qmake,cl.exe -ErrorAction SilentlyContinue | Select-Object Name,Source"` records available tools after bootstrap; Evidence `.omo/evidence/qt-rust-skeleton-completion/t1-toolchain.json`. Failure: same command in the current known shell records missing Qt/Rust/MSVC tools and blocks build/run work without changing product files; Evidence `.omo/evidence/qt-rust-skeleton-completion/t1-blocker.md`.
  Commit: N | baseline evidence only

- [x] T2. Fix preflight diagnostics and preserve portable bootstrap semantics
  What to do / Must NOT do: Narrowly fix `scripts/check-qt-rust-env.ps1` so empty `QT_ROOT`, `CMAKE_PREFIX_PATH`, or `PORTWEAVE_QT_RUST_ENV_SCRIPT` produce intended friendly messages instead of PowerShell null-argument errors. Keep the private bootstrap pattern portable and do not embed local paths.
  Parallelization: Wave 1 | Blocked by: T1 | Blocks: T3, T4, T5, T6, T8, T9, T10
  References (executor has NO interview context - be exhaustive): `scripts/check-qt-rust-env.ps1`; `Makefile` targets `qt-rust-env-check` and `migration-preflight`; `docs/development/qt-rust-skeleton-startup.md`; `docs/development/migration-gap-report.md`; memory note that shared docs must use `PATH`, `QT_ROOT`, `CMAKE_PREFIX_PATH`, and `PORTWEAVE_QT_RUST_ENV_SCRIPT`
  Acceptance criteria (agent-executable): Negative preflight exits nonzero with a concise missing-prerequisite message and no PowerShell exception; positive preflight passes when a valid private bootstrap/toolchain is available; docs remain free of local machine paths.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -ExecutionPolicy Bypass -File scripts\check-qt-rust-env.ps1` after bootstrap prints `Qt/Rust environment preflight OK`; Evidence `.omo/evidence/qt-rust-skeleton-completion/t2-preflight-positive.txt`. Failure: `powershell -NoProfile -Command "Remove-Item Env:QT_ROOT,Env:CMAKE_PREFIX_PATH,Env:PORTWEAVE_QT_RUST_ENV_SCRIPT -ErrorAction SilentlyContinue; & .\scripts\check-qt-rust-env.ps1"` exits nonzero with a friendly missing-env message; Evidence `.omo/evidence/qt-rust-skeleton-completion/t2-preflight-negative.txt`.
  Commit: Y | `fix(qt-rust): make environment preflight diagnostics deterministic`

- [ ] T3. Add failing-first QML feature/action matrix gate
  What to do / Must NOT do: Extend `scripts/check-migration-matrix.py` and its tests/fixtures so actionable QML controls are gated now that QML skeleton files exist. Allow `shell.smoke.new`, `shell.smoke.open`, `shell.smoke.save`, `shell.smoke.save_as`, and pure module navigation IDs such as `serial` and `settings`; reject uncontracted `workspace.*`, `serial.*`, `graph.*`, `virtual.*`, `mcp.*`, and `settings.theme` action IDs unless the matrix row is explicitly contracted with matching tests. Do not mark first-wave product rows implemented because a placeholder control exists.
  Parallelization: Wave 1 | Blocked by: T1, T10 | Blocks: T8, T9
  References (executor has NO interview context - be exhaustive): `scripts/check-migration-matrix.py`; `docs/development/migration-feature-matrix.csv`; `docs/development/qml-first-wave-design.md`; `docs/development/migration-readiness.md`; `desktop/qt-rust/qml/shell/MainToolbar.qml`; `desktop/qt-rust/qml/shell/MainWindow.qml`; `desktop/qt-rust/qml/shell/ActivityBar.qml`; `desktop/qt-rust/qml/shell/EditorGroups.qml`
  Acceptance criteria (agent-executable): Checker passes the current skeleton QML because controls remain skeleton-only; checker fails on a deliberate fixture/temp copy containing an actionable uncontracted product feature ID; first-wave rows remain `contracted`/not implemented unless all matrix requirements are present.
  QA scenarios (name the exact tool + invocation): Happy: `python scripts\check-migration-matrix.py > .omo\evidence\qt-rust-skeleton-completion\t3-matrix-positive.txt 2>&1` exits 0 on the repo. Failure: `powershell -NoProfile -Command "New-Item -ItemType Directory -Force .omo\tmp\qt-rust-negative-qml | Out-Null; Copy-Item desktop\qt-rust\qml\shell\MainToolbar.qml .omo\tmp\qt-rust-negative-qml\MainToolbar.qml; Add-Content .omo\tmp\qt-rust-negative-qml\MainToolbar.qml 'Button { property string featureId: \"workspace.open\" }'; python scripts\check-migration-matrix.py --qml-root .omo\tmp\qt-rust-negative-qml *> .omo\evidence\qt-rust-skeleton-completion\t3-matrix-negative.txt; if ($LASTEXITCODE -eq 0) { throw 'negative QML featureId fixture unexpectedly passed' }"` exits 0 only when the checker rejects the offending `workspace.open` fixture; Evidence `.omo/evidence/qt-rust-skeleton-completion/t3-matrix-negative.txt`.
  Commit: Y | `test(qt-rust): gate QML skeleton actions against migration matrix`

- [ ] T4. Prove Rust smoke library independently and keep the ABI narrow
  What to do / Must NOT do: Add or adjust Rust tests only as needed to cover exported smoke values, error labels/kinds, null/invalid boundary behavior if exposed, static string lifetime expectations, and UTF-8 safety. Keep Rust independent from Qt/QML and keep the C ABI surface narrow.
  Parallelization: Wave 2 | Blocked by: T2, T10 | Blocks: T5, T8, T9
  References (executor has NO interview context - be exhaustive): `desktop/qt-rust/rust/Cargo.toml`; `desktop/qt-rust/rust/src/lib.rs`; `desktop/qt-rust/rust/src/event.rs`; `docs/development/architecture-decisions.md` ADR-0006; `docs/development/qt-rust-skeleton-startup.md`
  Acceptance criteria (agent-executable): Rust tests pass without Qt installed; grep boundary check shows no Qt/QML dependency in Rust source/manifest; exported C ABI remains limited to skeleton smoke/result/error primitives.
  QA scenarios (name the exact tool + invocation): Happy: `cargo test --manifest-path desktop\qt-rust\rust\Cargo.toml > .omo\evidence\qt-rust-skeleton-completion\t4-cargo-test.txt 2>&1` passes. Failure: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "Qt|QML|QObject|qmetaobject|cxx_qt" -Path desktop/qt-rust/rust/src,desktop/qt-rust/rust/Cargo.toml -OutFile .omo\evidence\qt-rust-skeleton-completion\t4-rust-boundary.txt` treats no matches as pass, forbidden matches as failure, and `rg` errors as tool errors; if matches appear, T4 fails and must remove the coupling.
  Commit: Y | `test(qt-rust): cover Rust smoke ABI boundary`

- [ ] T5. Add Qt/C++ test harness and CTest coverage for bridge/lifecycle seams
  What to do / Must NOT do: Add `enable_testing()`/CTest wiring and minimal Qt tests for `AppBridge` and `WindowLifecycleProbe` around existing seams. Keep tests scoped to skeleton state, lifecycle event recording, and bridge smoke values. Do not introduce product ViewModel behavior or real workspace/serial/graph state.
  Parallelization: Wave 2 | Blocked by: T2, T3, T4 | Blocks: T6, T8, T9
  References (executor has NO interview context - be exhaustive): `desktop/qt-rust/CMakeLists.txt`; `desktop/qt-rust/app/AppBridge.cpp`; `desktop/qt-rust/app/AppBridge.h`; `desktop/qt-rust/app/WindowLifecycleProbe.cpp`; `desktop/qt-rust/app/WindowLifecycleProbe.h`; `desktop/qt-rust/app/main.cpp`; `desktop/qt-rust/README.md`
  Acceptance criteria (agent-executable): CMake configures tests, `ctest --test-dir build\qt-rust --output-on-failure` runs bridge/lifecycle tests, and no test requires human interaction.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -NoProfile -Command "cmake -S desktop\qt-rust -B build\qt-rust -G Ninja -DCMAKE_BUILD_TYPE=Debug; if($LASTEXITCODE){exit $LASTEXITCODE}; cmake --build build\qt-rust; if($LASTEXITCODE){exit $LASTEXITCODE}; ctest --test-dir build\qt-rust --output-on-failure" > .omo\evidence\qt-rust-skeleton-completion\t5-ctest.txt 2>&1` passes. Failure: `powershell -NoProfile -Command "$env:PORTWEAVE_QT_RUST_FORCE_LIFECYCLE_TEST_FAILURE='1'; ctest --test-dir build\qt-rust --output-on-failure -R WindowLifecycle *> .omo\evidence\qt-rust-skeleton-completion\t5-ctest-negative.txt; if($LASTEXITCODE -eq 0){ throw 'forced lifecycle negative test unexpectedly passed' }"` exits 0 only when the explicit forced-negative lifecycle test fails with the relevant assertion; Evidence `.omo/evidence/qt-rust-skeleton-completion/t5-ctest-negative.txt`.
  Commit: Y | `test(qt-rust): add Qt bridge and lifecycle CTest coverage`

- [ ] T6. Prove QML static health and app startup smoke
  What to do / Must NOT do: Add QML static lint/check target when Qt tooling is available, ensure QML is copied beside the executable, and verify `--smoke-exit` proves the app can initialize bridge/QML and exit cleanly. Do not count process survival alone if QML root loading fails silently.
  Parallelization: Wave 2 | Blocked by: T5, T10 | Blocks: T7, T8, T9
  References (executor has NO interview context - be exhaustive): `desktop/qt-rust/CMakeLists.txt`; `desktop/qt-rust/app/main.cpp`; `desktop/qt-rust/qml/shell/MainWindow.qml`; `desktop/qt-rust/qml/shell/StatusBar.qml`; `desktop/qt-rust/qml/shell/EditorGroups.qml`; `docs/development/qt-rust-skeleton-startup.md`
  Acceptance criteria (agent-executable): Build output contains `PortWeaveQtRust`; QML files are available to the executable; `PortWeaveQtRust.exe --smoke-exit` exits 0 only after QML root object creation and bridge initialization succeed; QML boundary grep shows no direct Rust/FFI/process calls.
  QA scenarios (name the exact tool + invocation): Happy: `.\build\qt-rust\PortWeaveQtRust.exe --smoke-exit > .omo\evidence\qt-rust-skeleton-completion\t6-smoke-exit.txt 2>&1` exits 0. Failure: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "portweave_rust_|extern|Library|ffi|Process" -Path desktop/qt-rust/qml -OutFile .omo\evidence\qt-rust-skeleton-completion\t6-qml-boundary.txt` treats no matches as pass, forbidden matches as failure, and `rg` errors as tool errors; if matches appear, T6 fails until removed.
  Commit: Y | `test(qt-rust): add QML and launch smoke gates`

- [ ] T7. Build repeatable Windows desktop GUI smoke evidence
  What to do / Must NOT do: Add a required repo-local PowerShell harness at `desktop/qt-rust/scripts/smoke-gui.ps1`. Launch the built app, locate window title `PortWeave Qt/Rust Skeleton`, capture screenshots, resize/minimize/restore, move across monitors when available, and record lifecycle/status observations. Do not require a human to click through the app; do not block forever if only one monitor exists.
  Parallelization: Wave 3 | Blocked by: T6 | Blocks: T9, T11
  References (executor has NO interview context - be exhaustive): `docs/development/testing.md`; `docs/development/qt-rust-skeleton-startup.md`; `docs/development/qt-rust-migration-playbook.md`; `desktop/qt-rust/app/main.cpp`; `desktop/qt-rust/app/WindowLifecycleProbe.cpp`; `desktop/qt-rust/qml/shell/MainWindow.qml`; `desktop/qt-rust/qml/shell/StatusBar.qml`; `desktop/qt-rust/qml/shell/EditorGroups.qml`
  Acceptance criteria (agent-executable): `desktop/qt-rust/scripts/smoke-gui.ps1` exists, exits 0 only after writing a receipt that lists PID, launch command, screenshots, monitor count, resize/minimize/restore actions, optional screen-move action or single-monitor fallback, final observed lifecycle/status text, cleanup result, and whether Rust smoke text was visible.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\smoke-gui.ps1 -Exe .\build\qt-rust\PortWeaveQtRust.exe -OutDir .omo\evidence\qt-rust-skeleton-completion\gui -ExpectedTitle "PortWeave Qt/Rust Skeleton" -RequireText "Rust smoke result:"` exits 0 and writes `.omo\evidence\qt-rust-skeleton-completion\gui\receipt.md` plus screenshots; Evidence `.omo/evidence/qt-rust-skeleton-completion/t7-gui-smoke.md`. Failure: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\smoke-gui.ps1 -Exe .\build\qt-rust\missing.exe -OutDir .omo\evidence\qt-rust-skeleton-completion\gui-negative -ExpectedTitle "PortWeave Qt/Rust Skeleton" -RequireText "Rust smoke result:"` exits nonzero with a clear missing-executable diagnostic; Evidence `.omo/evidence/qt-rust-skeleton-completion/t7-gui-smoke-negative.txt`.
  Commit: Y | `test(qt-rust): add repeatable desktop lifecycle smoke`

- [ ] T8. Add CI only after local Qt/Rust build proof exists
  What to do / Must NOT do: Update `.github/workflows/ci.yml` with the smallest safe Qt/Rust/migration coverage after T3-T6 pass locally. At minimum run `make migration-preflight`; add configure/build/test steps only in an environment that explicitly installs or exposes Qt/Rust/MSVC/Ninja through portable CI setup. Do not edit release workflow for this and do not add CI that can only pass on the local machine.
  Parallelization: Wave 4 | Blocked by: T3, T4, T5, T6, T10 | Blocks: T9, T11
  References (executor has NO interview context - be exhaustive): `.github/workflows/ci.yml`; `Makefile` targets `qt-rust-env-check`, `migration-matrix-check`, `migration-preflight`; `scripts/check-qt-rust-env.ps1`; `scripts/check-migration-matrix.py`; `desktop/qt-rust/CMakeLists.txt`; `docs/development/migration-gap-report.md`
  Acceptance criteria (agent-executable): CI job is adjacent to existing CI convention, uses portable setup, runs migration matrix/preflight, and conditionally includes Qt/Rust configure/build/test only when CI installs or exposes required toolchain. Workflow YAML validates structurally and local equivalent commands pass.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -NoProfile -Command "make migration-preflight; if($LASTEXITCODE){exit $LASTEXITCODE}; cargo test --manifest-path desktop\qt-rust\rust\Cargo.toml; if($LASTEXITCODE){exit $LASTEXITCODE}; cmake -S desktop\qt-rust -B build\qt-rust -G Ninja -DCMAKE_BUILD_TYPE=Debug; if($LASTEXITCODE){exit $LASTEXITCODE}; cmake --build build\qt-rust; if($LASTEXITCODE){exit $LASTEXITCODE}; ctest --test-dir build\qt-rust --output-on-failure" > .omo\evidence\qt-rust-skeleton-completion\t8-ci-local-equivalent.txt 2>&1` passes before CI build steps are committed. Failure: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "D:\\|C:\\Users\\|PORTWEAVE_QT_RUST_ENV_SCRIPT=.*[A-Za-z]:" -Path .github/workflows/ci.yml -OutFile .omo\evidence\qt-rust-skeleton-completion\t8-ci-path-scan.txt` treats no matches as pass and any local path/bootstrap value in CI as failure.
  Commit: Y | `ci(qt-rust): add migration skeleton verification`

- [ ] T9. Update migration docs with actual skeleton evidence and blockers
  What to do / Must NOT do: Update `qt-rust-skeleton-startup.md` and `migration-gap-report.md` so stale claims are corrected: skeleton files now exist, prior toolchain-pass claims are not treated as current unless revalidated, CI status reflects T8, and any missing toolchain/GUI limitation is explicitly recorded. Keep docs portable and evidence-based.
  Parallelization: Wave 4 | Blocked by: T6, T7, T8, T10 | Blocks: T11, final verification
  References (executor has NO interview context - be exhaustive): `docs/development/qt-rust-skeleton-startup.md`; `docs/development/migration-gap-report.md`; `docs/development/migration-readiness.md`; `docs/development/architecture-decisions.md` ADR-0006; all evidence files from T1-T8
  Acceptance criteria (agent-executable): Docs state the current skeleton completion evidence, remaining blockers, Rust decision-gate status, and CI status without machine-local paths; docs do not claim product migration readiness beyond skeleton gates.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "skeleton project files do not exist|Qt/Rust/MSVC are available|toolchain smokes passed" -Path docs/development/migration-gap-report.md,docs/development/qt-rust-skeleton-startup.md -OutFile .omo\evidence\qt-rust-skeleton-completion\t9-stale-doc-scan.txt` treats no stale claims as pass and any stale claim as failure. Failure: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "D:\\|C:\\Users\\|PORTWEAVE_QT_RUST_ENV_SCRIPT=.*[A-Za-z]:" -Path docs/development,scripts -OutFile .omo\evidence\qt-rust-skeleton-completion\t9-path-leak-scan.txt` treats no local path leaks as pass and any local path/bootstrap value as failure.
  Commit: Y | `docs(qt-rust): record skeleton completion evidence`

- [ ] T10. Add shared and final verification helper scripts
  What to do / Must NOT do: Create `desktop/qt-rust/scripts/assert-no-rg-match.ps1`, `desktop/qt-rust/scripts/final-plan-compliance.ps1`, and `desktop/qt-rust/scripts/final-scope-fidelity.ps1` before any todo invokes those helpers. The no-match helper must treat `rg` exit code 1 with empty output as pass, exit code 0 with forbidden matches as failure, and exit code greater than 1 as tool error. The final compliance script must verify expected evidence or explicit blocker files and write its declared output; `-AllowBlockers` is only for T10 smoke mode, where missing downstream evidence is written as blocker text without claiming final pass. The final scope script must run the product-scope, QML-boundary, Rust-boundary, and path-leak scans, handle all no-match pass cases explicitly, and write `final-f4-scope-fidelity.md`; `-AllowBlockers` is only for T10 smoke mode. Do not create a generic test framework or reusable utility outside `desktop/qt-rust/scripts/`.
  Parallelization: Wave 1 | Blocked by: T2 | Blocks: T3, T4, T6, T8, T9, T11, final verification
  References (executor has NO interview context - be exhaustive): Final verification wave F1-F4 in this plan; planned T7 GUI-smoke command path `desktop/qt-rust/scripts/smoke-gui.ps1`; boundary scan commands in T4, T6, T8, and T9; repository `AGENTS.md` verification and dirty-worktree rules.
  Acceptance criteria (agent-executable): All three scripts exist, accept the parameters shown in this plan, create parent directories for output files, use explicit exit codes, and fail with clear diagnostics for forbidden matches, missing evidence, or `rg` tool errors.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern "definitely_not_present_qt_rust_skeleton_marker" -Path desktop/qt-rust -OutFile .omo\evidence\qt-rust-skeleton-completion\t10-no-rg-match-positive.txt` exits 0 and writes a pass receipt; `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\final-plan-compliance.ps1 -Plan .omo\plans\qt-rust-skeleton-completion.md -EvidenceRoot .omo\evidence\qt-rust-skeleton-completion -OutFile .omo\evidence\qt-rust-skeleton-completion\t10-final-plan-compliance-smoke.txt -AllowBlockers` exits 0 in smoke mode and writes missing downstream evidence as blockers; `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\final-scope-fidelity.ps1 -EvidenceRoot .omo\evidence\qt-rust-skeleton-completion -OutFile .omo\evidence\qt-rust-skeleton-completion\t10-final-scope-fidelity-smoke.txt -AllowBlockers` exits 0 in smoke mode and writes blocker text instead of claiming final pass; `powershell -NoProfile -Command "'final helper positive checks passed' | Set-Content .omo\evidence\qt-rust-skeleton-completion\t10-final-helper-positive.txt"` writes the declared combined receipt; Evidence `.omo/evidence/qt-rust-skeleton-completion/t10-final-helper-positive.txt`. Failure: `powershell -NoProfile -Command "New-Item -ItemType Directory -Force .omo\tmp\qt-rust-rg-negative | Out-Null; 'FORBIDDEN_QT_RUST_HELPER_MARKER' | Set-Content .omo\tmp\qt-rust-rg-negative\forbidden.txt; powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\assert-no-rg-match.ps1 -Pattern 'FORBIDDEN_QT_RUST_HELPER_MARKER' -Path .omo\tmp\qt-rust-rg-negative -OutFile .omo\evidence\qt-rust-skeleton-completion\t10-no-rg-match-negative.txt; if($LASTEXITCODE -eq 0){ throw 'forbidden marker unexpectedly passed no-match helper' }"` exits 0 only when the helper rejects the fixture marker; Evidence `.omo/evidence/qt-rust-skeleton-completion/t10-no-rg-match-negative.txt`.
  Commit: Y | `test(qt-rust): add final verification helpers`

- [ ] T11. Prepare narrow staging and final verification receipt
  What to do / Must NOT do: Review diff scope, ensure unrelated dirty files remain untouched/untracked, run final checks, and prepare a narrow commit grouping. Do not stage `.hermes/`, `third_party/`, unrelated `go.mod/go.sum`, `.omo/evidence`, or any scratch artifacts.
  Parallelization: Wave 5 | Blocked by: T9, T10 | Blocks: final verification
  References (executor has NO interview context - be exhaustive): `git status --short --branch`; all touched files from T2-T10; repository `AGENTS.md` instructions; `.omo/drafts/qt-rust-skeleton-completion.md`
  Acceptance criteria (agent-executable): `git diff --check` passes; status receipt shows only intended task files changed/staged; evidence index lists all commands run and any skipped checks with concrete reasons; commit list is atomic and excludes unrelated local state.
  QA scenarios (name the exact tool + invocation): Happy: `powershell -NoProfile -Command "git diff --check; if($LASTEXITCODE){exit $LASTEXITCODE}; git status --short --branch | Set-Content .omo\evidence\qt-rust-skeleton-completion\t11-final-status.txt"` passes/records narrow scope. Failure: `powershell -NoProfile -Command "$bad = git diff --name-only --cached | Where-Object { $_ -match '^(go\.mod|go\.sum|\.hermes/|third_party/|\.omo/evidence/)' }; $bad | Set-Content .omo\evidence\qt-rust-skeleton-completion\t11-staging-guard.txt; if($bad){ throw 'forbidden staged files: ' + ($bad -join ', ') }"` exits nonzero if forbidden files are staged; Evidence `.omo/evidence/qt-rust-skeleton-completion/t11-staging-guard.txt`.
  Commit: Y | final commits only after T11 verifies the stage set

## Final verification wave
> Runs in parallel after ALL todos. ALL must APPROVE. Surface results and wait for the user's explicit okay before declaring complete.
- [ ] F1. Plan compliance audit
  Invocation: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\final-plan-compliance.ps1 -Plan .omo\plans\qt-rust-skeleton-completion.md -EvidenceRoot .omo\evidence\qt-rust-skeleton-completion -OutFile .omo\evidence\qt-rust-skeleton-completion\final-f1-plan-compliance.txt`
  Pass condition: every completed change maps to T1-T11, required evidence exists or an explicit blocker file exists, and no product feature migration is present. Evidence `.omo/evidence/qt-rust-skeleton-completion/final-f1-plan-compliance.txt`.
- [ ] F2. Code quality review
  Invocation: `powershell -NoProfile -Command "git diff -- . ':!.omo/evidence/**' | Out-File .omo\evidence\qt-rust-skeleton-completion\final-f2-diff.patch; git diff --check 2>&1 | Tee-Object .omo\evidence\qt-rust-skeleton-completion\final-f2-diff-check.txt; if($LASTEXITCODE){exit $LASTEXITCODE}; git diff --name-only | Tee-Object .omo\evidence\qt-rust-skeleton-completion\final-f2-files.txt"`
  Pass condition: reviewer inspects the recorded diff for changed scripts, tests, CMake, Qt/C++/QML, Rust, CI, and docs; `git diff --check` exits 0; no broad abstraction or speculative migration work is present. Evidence `.omo/evidence/qt-rust-skeleton-completion/final-f2-diff-check.txt`.
- [ ] F3. Real desktop QA
  Invocation: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\smoke-gui.ps1 -Exe .\build\qt-rust\PortWeaveQtRust.exe -OutDir .omo\evidence\qt-rust-skeleton-completion\final-gui -ExpectedTitle "PortWeave Qt/Rust Skeleton" -RequireText "Rust smoke result:"`
  Pass condition: command exits 0; `final-gui\receipt.md` confirms visible QML, Rust smoke result, lifecycle/status text, resize/minimize/restore, cleanup, and monitor move or single-monitor fallback. Evidence `.omo/evidence/qt-rust-skeleton-completion/final-gui/receipt.md`.
- [ ] F4. Scope fidelity
  Invocation: `powershell -ExecutionPolicy Bypass -File desktop\qt-rust\scripts\final-scope-fidelity.ps1 -EvidenceRoot .omo\evidence\qt-rust-skeleton-completion -OutFile .omo\evidence\qt-rust-skeleton-completion\final-f4-scope-fidelity.md`
  Pass condition: scans show only allowed `shell.smoke.*`/navigation skeleton scope or documented contracted rows; QML has no Rust/FFI/process coupling; Rust has no Qt/QML coupling; shared docs/scripts have no local path leaks. Evidence `.omo/evidence/qt-rust-skeleton-completion/final-f4-scope-fidelity.md`.

## Commit strategy
- Commit 1: preflight and verification helpers (`scripts/check-qt-rust-env.ps1`, `desktop/qt-rust/scripts/assert-no-rg-match.ps1`, `desktop/qt-rust/scripts/final-plan-compliance.ps1`, `desktop/qt-rust/scripts/final-scope-fidelity.ps1`).
- Commit 2: matrix guardrails (`scripts/check-migration-matrix.py`, tests/fixtures).
- Commit 3: Rust/Qt/QML skeleton verification harnesses (`desktop/qt-rust/**` tests, CTest, QML/app smoke support).
- Commit 4: desktop GUI smoke harness if a repo-local script/runbook is added.
- Commit 5: CI migration skeleton job after local proof.
- Commit 6: documentation evidence updates.
- Keep `.omo/evidence`, `.hermes/`, `third_party/`, and unrelated dirty files uncommitted unless the user explicitly changes the scope.
- If the resulting diff is small after implementation, combine adjacent commits only when the combined commit remains reviewable and atomic.

## Success criteria
- Current branch remains `feat/qt-rust-merged` unless the user explicitly changes branch strategy.
- Environment preflight has deterministic positive/negative behavior and no null-argument PowerShell errors.
- Migration matrix checker gates QML actions/feature IDs and catches uncontracted product-surface controls.
- Rust smoke library passes `cargo test` without Qt and remains free of Qt/QML dependencies.
- CMake configures/builds the Qt/Rust executable with Ninja, CTest runs bridge/lifecycle checks, and `PortWeaveQtRust.exe --smoke-exit` exits 0 after QML initialization.
- Desktop GUI evidence proves visible QML, visible Rust smoke result, lifecycle/status observability, resize/minimize/restore behavior, monitor move or documented single-monitor fallback, and clean process shutdown.
- CI covers migration preflight and, after local proof, the Qt/Rust build/test gates using portable setup.
- Docs reflect actual current evidence and blockers, not stale claims that files are missing or tools already passed.
- `git diff --check` passes and final status/staging excludes unrelated local changes.
- `desktop\qt-rust\scripts\assert-no-rg-match.ps1`, `desktop\qt-rust\scripts\final-plan-compliance.ps1`, and `desktop\qt-rust\scripts\final-scope-fidelity.ps1` exist, are executable from PowerShell, handle `rg` no-match pass cases explicitly, and write the declared final evidence files.
