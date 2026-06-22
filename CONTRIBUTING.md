# Contributing

MockTrue is a Wails v3 desktop application with a Go backend and a Vue/TypeScript frontend.

## Development

Install the Wails CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.103
```

Install frontend dependencies:

```bash
cd frontend
pnpm install
```

Run the app from the repository root:

```bash
wails3 dev -config ./build/config.yml -port 9245
```

## Checks

Run the checks that match the files you changed. For frontend changes:

```bash
cd frontend
pnpm test -- --run
pnpm exec vue-tsc --noEmit
pnpm run build:dev
```

For backend serial changes:

```bash
go test ./internal/modules/serial/... -count=1
```

For broader backend changes:

```bash
go test ./... -count=1
```

Optional Windows real-COM integration uses a preconfigured com0com pair; see [Windows Serial Testing](docs/development/windows-serial-testing.md).

## Commit Guidelines

Use focused commits and Conventional Commit messages, for example:

```text
docs: update public project documentation
```

Do not include generated output, local logs, credentials, or machine-specific files in commits.
