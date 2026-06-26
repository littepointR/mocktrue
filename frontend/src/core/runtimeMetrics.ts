import { Snapshot } from '../../bindings/github.com/littepointR/portweave/internal/core/runtime/service.js'
import type { Metrics } from '../../bindings/github.com/littepointR/portweave/internal/core/runtime/models.js'

export type RuntimeMetrics = Metrics

export async function getRuntimeMetrics(): Promise<RuntimeMetrics> {
  if (hasWailsBackend()) {
    try {
      const metrics = await Snapshot()
      if (metrics) return metrics
    } catch {
      // Static browser previews do not have the Wails backend. Fall back to
      // browser-visible metrics so the status bar still has live values.
    }
  }

  return browserRuntimeMetrics()
}

function hasWailsBackend(): boolean {
  const wails = (window as Window & {
    _wails?: { environment?: unknown }
    __mockState?: unknown
  })
  return Boolean(wails._wails?.environment || wails.__mockState)
}

function browserRuntimeMetrics(): RuntimeMetrics {
  return {
    CPUPercent: 0,
    MemoryBytes: performanceMemory(),
  }
}

function performanceMemory(): number {
  const memory = (performance as Performance & {
    memory?: {
      usedJSHeapSize?: number
      totalJSHeapSize?: number
    }
  }).memory

  return memory?.usedJSHeapSize ?? memory?.totalJSHeapSize ?? 0
}
