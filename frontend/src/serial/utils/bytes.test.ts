import { describe, expect, it } from 'vitest'
import { toByteArray } from './bytes'

describe('toByteArray', () => {
  it('keeps numeric byte arrays unchanged', () => {
    expect(toByteArray([65, 66, 67])).toEqual([65, 66, 67])
  })

  it('converts Uint8Array values to number arrays', () => {
    expect(toByteArray(new Uint8Array([0, 255, 16]))).toEqual([0, 255, 16])
  })

  it('decodes base64 strings emitted for Go byte slices', () => {
    expect(toByteArray('dGVzdCBkYXRh')).toEqual([116, 101, 115, 116, 32, 100, 97, 116, 97])
  })

  it('treats missing data as empty', () => {
    expect(toByteArray(null)).toEqual([])
    expect(toByteArray(undefined)).toEqual([])
  })
})
