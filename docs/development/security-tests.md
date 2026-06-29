# Security Tests

This page defines the current PortWeave security-test taxonomy and release gates. It focuses on local automation, serial resources, raw TCP, script execution, and artifact hygiene.

## Security boundaries

PortWeave is local-first desktop/server-mode software. The risky boundaries are:

- MCP tools that can create resources, send bytes, clear/delete state, or operate devices.
- Raw TCP serial nodes, which have no built-in authentication, authorization, encryption, or serial parameter negotiation.
- Script runtimes executing operator-provided JavaScript.
- Imports/exports, screenshots, logs, and docs that might accidentally include credentials.
- Release artifacts and checksums.

## Test taxonomy

### MCP safety annotations and tool inventory

Coverage goals:

- Read-only catalog/query tools are marked read-only.
- Create/start/send/delete/clear/reset tools are not marked read-only.
- Docs-critical tools stay present in `tools/list`.
- New public tools have tests and are documented before release.

Current targeted command:

```bash
go test ./internal/modules/mcpserver -run 'TestRegisterTools|TestProtocolTemplateMCPTools|TestMCPToolAnnotations' -count=1
```

### Raw TCP and remote serial safety

Coverage goals:

- Remote endpoint validation rejects invalid host/port/protocol/role combinations.
- Duplicate resource ownership checks include normalized raw TCP endpoint plus role.
- Docs state that raw TCP is trusted-network only.
- Tests avoid public network endpoints.

### Script runtime containment

Coverage goals:

- Empty scripts fail.
- Unsupported encodings fail.
- Invalid hex/byte arrays fail.
- Output limits are enforced.
- State-size limits are enforced.
- Execution timeouts are enforced.
- Analyzer errors are recorded as data evidence instead of crashing the process.

Current targeted command:

```bash
go test ./internal/modules/serial -run 'TestSerialScriptRuntime' -count=1
```

### Protocol/parser robustness

Coverage goals:

- Visual parser length and checksum errors are data errors, not panics.
- Parser template examples are executable or clearly documented as compatibility fixtures.
- Script parser templates do not accidentally use graph-node APIs.

Current targeted command:

```bash
go test ./internal/modules/serial/protocol/... -count=1
```

### Secrets and artifact hygiene

Before release or handoff, inspect changed docs/workflows/source for obvious secret markers. If operator-provided context contains secrets, redact them as `[REDACTED]`.

Suggested local check for changed files:

```bash
git diff -- docs .github/workflows internal frontend \
  | grep -Ei 'api[_-]?key|token|secret|password|private[_-]?key|connection string' || true
```

A match is not automatically a failure; examples may mention these words. Review every match and ensure no real credential material is present.

## Required release security gates

Before publishing a release:

1. `git diff --check` passes on release-relevant files.
2. Targeted MCP/serial/protocol tests pass.
3. Coverage gates pass unless the release notes explicitly identify an emergency exception.
4. Release artifacts are smoke-checked and checksummed.
5. Raw TCP/MCP docs continue to state trusted-local-network assumptions.
6. No generated binding diffs are present unless exported Go service/model signatures changed and the diffs were reviewed.

## Non-goals for this phase

- No public MCP hosting model.
- No built-in TLS/VPN/auth layer for raw TCP serial.
- No automatic writes/scans against physical devices in default tests.
- No release matrix expansion without artifact smoke evidence.

## Related docs

- [MCP API and Recipes](../mcp-api.md)
- [AI Automation Guide](../ai-automation.md)
- [Headless Integration](headless-integration.md)
- [Release](release.md)
