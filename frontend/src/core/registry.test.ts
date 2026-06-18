import { describe, it, expect, beforeEach } from 'vitest'
import { useRegistry, __resetRegistryForTest } from './registry'
import type { ModuleFrontend } from './module/types'

function makeModule(id: string): ModuleFrontend & { activated: number; deactivated: number } {
  const m = {
    id,
    activity: { icon: id, title: id },
    views: [{ id: `${id}.v`, title: 'v', component: `${id}/V` }],
    activated: 0,
    deactivated: 0,
  } as ModuleFrontend & { activated: number; deactivated: number }
  m.onActivate = () => { m.activated++ }
  m.onDeactivate = () => { m.deactivated++ }
  return m
}

describe('ModuleRegistry', () => {
  beforeEach(() => __resetRegistryForTest())

  it('registers and lists a module', () => {
    const r = useRegistry()
    r.register(makeModule('serial'))
    expect(r.list()).toHaveLength(1)
    expect(r.list()[0].moduleId).toBe('serial')
  })

  it('rejects duplicate id', () => {
    const r = useRegistry()
    r.register(makeModule('serial'))
    expect(() => r.register(makeModule('serial'))).toThrow()
  })

  it('rejects empty id', () => {
    const r = useRegistry()
    expect(() => r.register(makeModule(''))).toThrow()
  })

  it('activates and deactivates modules, firing hooks', () => {
    const r = useRegistry()
    const a = makeModule('a')
    const b = makeModule('b')
    r.register(a)
    r.register(b)

    r.setActive('a')
    expect(r.activeModule()?.id).toBe('a')
    expect(a.activated).toBe(1)

    r.setActive('b')
    expect(r.activeModule()?.id).toBe('b')
    expect(a.deactivated).toBe(1)
    expect(b.activated).toBe(1)
  })

  it('throws on unknown id', () => {
    const r = useRegistry()
    expect(() => r.setActive('ghost')).toThrow()
  })

  it('allows deactivation via null', () => {
    const r = useRegistry()
    const a = makeModule('a')
    r.register(a)
    r.setActive('a')
    r.setActive(null)
    expect(r.activeModule()).toBeNull()
    expect(a.deactivated).toBe(1)
  })

  it('mergeBackendContributions tolerates null and bad shapes', () => {
    const r = useRegistry()
    r.register(makeModule('serial'))
    expect(() => r.mergeBackendContributions(null)).not.toThrow()
    expect(() => r.mergeBackendContributions([{ ModuleID: 'serial' }])).not.toThrow()
    expect(() => r.mergeBackendContributions([{ garbage: true }])).not.toThrow()
  })

  it('exposes a reactive active id', () => {
    const r = useRegistry()
    r.register(makeModule('serial'))
    expect(r.active.value).toBeNull()
    r.setActive('serial')
    expect(r.active.value).toBe('serial')
  })
})
