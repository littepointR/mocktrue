# Windows Serial Testing

MockTrue's default test suite does not require real or virtual COM ports. The production serial backend still uses the Windows serial APIs through `go.bug.st/serial`, while default tests use deterministic in-memory ports where possible.

## Default checks

Run these without installing com0com or any serial hardware:

```powershell
go test ./... -count=1
cd frontend
pnpm test -- --run
pnpm exec vue-tsc --noEmit
pnpm run build:dev
```

The repository also supports the Makefile target:

```powershell
make test
```

`make test` includes integration packages. On Windows, POSIX socat tests are excluded by build tags; the Windows COM integration test skips unless the environment variables below are set.

## Optional real COM integration test

Install and configure a linked virtual COM pair with com0com, for example `COM7` <-> `COM8`.

Then run:

```powershell
$env:MOCKTRUE_TEST_COM_A = "COM7"
$env:MOCKTRUE_TEST_COM_B = "COM8"
go test -tags integration ./tests/go/integration -run TestWindowsVirtualCOMPairRoundTrip -count=1 -v
```

If the environment variables are not set, the test skips. This test is intentionally opt-in because com0com may require administrator privileges, driver installation, and machine-specific setup.

## POSIX virtual serial tests

macOS/Linux socat-backed integration tests are compiled only with:

```bash
go test -tags integration ./tests/go/integration ./tests/automation/integration -count=1 -v
```

They are excluded from Windows builds by `//go:build (darwin || linux) && integration`.

## Wails dev PATH on Windows

`wails3 dev` starts child processes that must also find `wails3`, `go`, `node`, and `pnpm` on `PATH`. If launching from Git Bash/MSYS, verify child-process PATH before debugging app code.

PowerShell example:

```powershell
$env:PATH = "$HOME\go\bin;C:\Program Files\Go\bin;D:\Program Files\nodejs;$env:PATH"
wails3 dev -config ./build/config.yml -port 9245
```

Git Bash example:

```bash
export PATH="/c/Users/pc/go/bin:/c/Program Files/Go/bin:/d/Program Files/nodejs:$PATH"
wails3 dev -config ./build/config.yml -port 9245
```

If port `9245` is already in use, either stop the existing dev server or use a different port:

```bash
wails3 dev -config ./build/config.yml -port 9255
```
