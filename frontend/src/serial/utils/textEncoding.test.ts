import { describe, expect, it } from 'vitest'
import { decodeSerialText, normalizeTextEncoding } from './textEncoding'

describe('textEncoding utilities', () => {
  it('normalizes empty and underscored encoding labels', () => {
    expect(normalizeTextEncoding('')).toBe('utf-8')
    expect(normalizeTextEncoding(' Shift_JIS ')).toBe('shift-jis')
  })

  it('decodes ascii bytes with replacement for non-ascii bytes', () => {
    expect(decodeSerialText(new Uint8Array([0x41, 0xff, 0x42]), 'ASCII')).toBe('A�B')
  })

  it('maps legacy browser encoding labels and falls back to utf-8 when decoding fails', () => {
    expect(decodeSerialText(new Uint8Array([0xc4, 0xe3]), 'gb2312')).toBe('你')
    expect(decodeSerialText(new Uint8Array([0x82, 0xa0]), 'shift-jis')).toBe('あ')
    expect(decodeSerialText(new Uint8Array([0x68, 0x69]), 'unsupported')).toBe('hi')
  })
})
