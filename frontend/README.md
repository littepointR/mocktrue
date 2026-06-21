# MockTrue Frontend

Vue 3, TypeScript, Vite, Naive UI, Pinia, and Monaco Editor frontend for the MockTrue Wails application.

## Development

Install dependencies from this directory:

```bash
pnpm install
```

Run frontend checks:

```bash
pnpm test -- --run
pnpm exec vue-tsc --noEmit
pnpm run build:dev
```

The desktop development entrypoint is the repository root Wails command documented in the root `README.md`.
