import { describe, expect, it } from 'vitest'
import { base64ToBytes, bytesToBase64, graphTabKind, stableStringify, workspaceKind } from './workspaceSnapshot'

describe('workspaceSnapshot utilities', () => {
  it('exports stable documented workspace and graph tab kind strings', () => {
    expect(workspaceKind).toBe('portweave.workspace.v1')
    expect(graphTabKind).toBe('portweave.graph.v1')
  })

  it('roundtrips arbitrary received bytes through base64', () => {
    const bytes = new Uint8Array([0, 1, 2, 127, 128, 255])
    expect(base64ToBytes(bytesToBase64(bytes))).toEqual(bytes)
  })

  it('serializes object keys in stable order for dirty tracking', () => {
    expect(stableStringify({ b: 2, a: { d: 4, c: 3 } })).toBe(stableStringify({ a: { c: 3, d: 4 }, b: 2 }))
    expect(stableStringify({ z: [{ b: 2, a: 1 }] })).toBe(stableStringify({ z: [{ a: 1, b: 2 }] }))
  })
})
