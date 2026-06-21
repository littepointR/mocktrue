import { vi } from 'vitest'

if (!document.queryCommandSupported) {
  document.queryCommandSupported = vi.fn(() => false)
}

vi.mock('@wailsio/runtime', () => ({
  Call: {
    ByID: vi.fn(async () => undefined),
  },
  CancellablePromise: Promise,
  Create: {
    Any: (value: any) => value,
    Array: (createItem: (value: any) => any) => (value: any[]) => Array.isArray(value) ? value.map(createItem) : [],
    ByteSlice: (value: any) => value ?? '',
    Nullable: (createValue: (value: any) => any) => (value: any) => value == null ? null : createValue(value),
  },
  Events: {
    Off: vi.fn(),
    On: vi.fn(() => vi.fn()),
  },
}))
