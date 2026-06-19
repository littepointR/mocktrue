import { vi } from 'vitest'

vi.mock('@wailsio/runtime', () => ({
  Call: {
    ByID: vi.fn(async () => undefined),
  },
  CancellablePromise: Promise,
  Events: {
    Off: vi.fn(),
    On: vi.fn(() => vi.fn()),
  },
}))
