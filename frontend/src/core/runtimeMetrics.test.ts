import { beforeEach, describe, expect, it, vi } from 'vitest'

const snapshotMock = vi.hoisted(() => vi.fn())

vi.mock('../../bindings/github.com/littepointR/mocktrue/internal/core/runtime/service.js', () => ({
  Snapshot: snapshotMock,
}))

import { getRuntimeMetrics } from './runtimeMetrics'

describe('runtimeMetrics', () => {
  beforeEach(() => {
    snapshotMock.mockReset()
    delete (window as any)._wails
    delete (window as any).__mockState
    Reflect.deleteProperty(performance, 'memory')
  })

  it('uses browser-visible heap memory when no Wails backend is present', async () => {
    Object.defineProperty(performance, 'memory', {
      configurable: true,
      value: { usedJSHeapSize: 1234, totalJSHeapSize: 5678 },
    })

    await expect(getRuntimeMetrics()).resolves.toEqual({
      CPUPercent: 0,
      MemoryBytes: 1234,
    })
    expect(snapshotMock).not.toHaveBeenCalled()
  })

  it('uses the backend snapshot when Wails runtime state is present', async () => {
    ;(window as any)._wails = { environment: { platform: 'darwin' } }
    snapshotMock.mockResolvedValue({ CPUPercent: 12.5, MemoryBytes: 4096 })

    await expect(getRuntimeMetrics()).resolves.toEqual({
      CPUPercent: 12.5,
      MemoryBytes: 4096,
    })
    expect(snapshotMock).toHaveBeenCalledTimes(1)
  })

  it('falls back to total heap memory when the backend snapshot is unavailable', async () => {
    ;(window as any).__mockState = {}
    snapshotMock.mockRejectedValue(new Error('not connected'))
    Object.defineProperty(performance, 'memory', {
      configurable: true,
      value: { totalJSHeapSize: 2048 },
    })

    await expect(getRuntimeMetrics()).resolves.toEqual({
      CPUPercent: 0,
      MemoryBytes: 2048,
    })
  })
})
