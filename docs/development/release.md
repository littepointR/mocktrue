# Release

This page defines the current PortWeave release discipline. It intentionally keeps the matrix narrow until artifact smoke checks are stable.

## Current release workflow

`.github/workflows/release.yml` runs for tags matching `v*.*.*` and for manual dispatch. The current workflow builds a macOS app bundle with Wails, archives it as `dist/portweave-macos.zip`, uploads it as a workflow artifact, and publishes it to a GitHub release only for tag builds.

Do not expand the release matrix until each new target has:

- A documented build command.
- Artifact existence checks.
- A minimal smoke check suitable for that platform.
- Checksums attached to the workflow artifacts/release.

## Local pre-release checklist

Run the relevant gates before creating a release tag:

```bash
make ci
make build
```

If strict local tooling is available, also run:

```bash
make ci-strict
```

`make lint` requires `golangci-lint`; report it as an environment blocker if missing rather than claiming lint passed.

For Wails/package changes, run the platform package command on the target OS. On macOS:

```bash
wails3 task darwin:package
```

## Artifact smoke discipline

Every packaged artifact should have a machine-checkable smoke step before upload:

- Verify the expected file exists and is non-empty.
- For zip artifacts, list archive contents.
- Ensure the expected `.app` or binary path is present.
- Generate a SHA-256 checksum file next to the artifact.
- Upload the artifact and checksum together.

For macOS zip artifacts, the smoke should verify at least:

```bash
test -s dist/portweave-macos.zip
unzip -l dist/portweave-macos.zip | grep -E 'portweave\.app/Contents/MacOS/portweave'
shasum -a 256 dist/portweave-macos.zip > dist/checksums.txt
grep 'portweave-macos.zip' dist/checksums.txt
```

This is not a substitute for a real GUI smoke. GUI status claims must verify both the native desktop process/window and the frontend/server as appropriate.

## Server/headless smoke discipline

Server-mode smoke is documented in [Headless Integration](headless-integration.md). Current confirmed behavior:

- `go build -tags server` succeeds locally.
- The server-mode process serves frontend HTML at `http://localhost:8080/`.
- MCP readiness still needs a real MCP HTTP transport check; `GET /mcp` is not a valid readiness test.

Do not add a release claim that MCP headless smoke passed until the harness actually performs MCP `tools/list` and safe graph validation through the running process.

## Versioning and notes

Before tagging:

1. Confirm `build/config.yml` version expectations.
2. Confirm README/docs examples still match source tool names and demo IDs.
3. Summarize verification commands and platform limitations.
4. Mention raw TCP/MCP trust assumptions when release notes highlight automation features.

## Checksums

The release workflow should attach checksums for every uploaded binary/archive. A single `dist/checksums.txt` is acceptable while the matrix has one artifact; expand it when the matrix expands.

## Rollback and cleanup

If artifact smoke fails:

- Do not publish a tag release.
- Keep failed workflow logs and artifact paths for diagnosis.
- Fix the packaging/smoke root cause before retrying.

If a tag release was published with a broken artifact, create a follow-up patch release instead of silently replacing artifacts without notes.

## Related docs

- [Testing](testing.md)
- [Benchmarks](benchmarks.md)
- [Security Tests](security-tests.md)
- [Headless Integration](headless-integration.md)
- [Architecture Decisions](architecture-decisions.md)
