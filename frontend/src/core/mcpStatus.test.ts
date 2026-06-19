import { describe, expect, it, vi } from 'vitest'

const off = vi.fn()
const on = vi.fn(() => off)

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: on,
  },
}))

describe('mcpStatus', () => {
  it('returns the exact Wails unsubscribe function', async () => {
    const { onMCPStatus } = await import('./mcpStatus')
    const callback = vi.fn()

    const cancel = onMCPStatus(callback)
    cancel()

    expect(on).toHaveBeenCalledWith('mcp:status', expect.any(Function))
    expect(off).toHaveBeenCalledTimes(1)
  })
})
