import { describe, expect, it } from 'vitest'
import {
  asciiToHexText,
  formatHexInput,
  hexTextToAscii,
  toByteArray,
} from './bytes'

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

describe('formatHexInput', () => {
  it('formats hex input as two-digit bytes separated by spaces', () => {
    expect(formatHexInput('48656c6C6f')).toBe('48 65 6c 6c 6f')
    expect(formatHexInput('48  65\n6C')).toBe('48 65 6c')
  })

  it('preserves a trailing half byte while the user is typing', () => {
    expect(formatHexInput('486')).toBe('48 6')
  })

  it('normalizes common pasted hex separators and 0x prefixes', () => {
    expect(formatHexInput('0x48,0x65-0x6c')).toBe('48 65 6c')
  })
})

describe('asciiToHexText', () => {
  it('encodes ascii text to formatted UTF-8 hex bytes', () => {
    expect(asciiToHexText('Hello')).toBe('48 65 6c 6c 6f')
    expect(asciiToHexText('串口')).toBe('e4 b8 b2 e5 8f a3')
  })
})

describe('hexTextToAscii', () => {
  it('decodes formatted hex bytes to text', () => {
    expect(hexTextToAscii('48 65 6c 6c 6f')).toBe('Hello')
    expect(hexTextToAscii('e4 b8 b2 e5 8f a3')).toBe('串口')
  })
})
