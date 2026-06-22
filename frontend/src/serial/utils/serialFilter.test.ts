import { describe, expect, it } from 'vitest'
import { matchSerialFilter, type SerialFilterCandidate, type SerialFilterOptions } from './serialFilter'

describe('serial filter matcher', () => {
  it('matches plain keywords with case-sensitive and whole-word options', () => {
    const candidate: SerialFilterCandidate = {
      text: 'Status OK on reporter',
      hex: '4f 4b',
      message: 'Gateway ready',
      level: 'info',
    }

    const cases: Array<{ name: string; options: SerialFilterOptions; matched: boolean }> = [
      { name: 'empty expression matches', options: { mode: 'plain', expression: '' }, matched: true },
      { name: 'case-insensitive keyword', options: { mode: 'plain', expression: 'ok' }, matched: true },
      { name: 'case-sensitive keyword', options: { mode: 'plain', expression: 'ok', caseSensitive: true }, matched: false },
      { name: 'substring without whole-word', options: { mode: 'plain', expression: 'port' }, matched: true },
      { name: 'whole-word rejects substring', options: { mode: 'plain', expression: 'port', wholeWord: true }, matched: false },
    ]

    for (const item of cases) {
      expect(matchSerialFilter(candidate, item.options), item.name).toEqual({ matched: item.matched })
    }
  })

  it('matches regex filters with case-sensitive and whole-word options and structured errors', () => {
    const candidate: SerialFilterCandidate = { text: 'TEMP=42 error', message: 'payload accepted' }

    expect(matchSerialFilter(candidate, { mode: 'regex', expression: 'temp=\\d+' })).toEqual({ matched: true })
    expect(matchSerialFilter(candidate, { mode: 'regex', expression: 'temp=\\d+', caseSensitive: true })).toEqual({ matched: false })
    expect(matchSerialFilter(candidate, { mode: 'regex', expression: 'err' })).toEqual({ matched: true })
    expect(matchSerialFilter(candidate, { mode: 'regex', expression: 'err', wholeWord: true })).toEqual({ matched: false })

    const invalid = matchSerialFilter(candidate, { mode: 'regex', expression: '[unterminated' })
    expect(invalid.matched).toBe(false)
    expect(invalid.error).toContain('invalid regex')
  })

  it('matches the documented Wireshark-like expression subset', () => {
    const candidate: SerialFilterCandidate = {
      len: 7,
      text: 'TEMP OK',
      hex: '0d 0a ff',
      message: 'template failed with fallback',
      level: 'error',
      source: 'runtime',
      graphId: 'graph-1',
      nodeId: 'rx-1',
      nodeType: 'serial.receiver',
      direction: 'rx',
    }

    const cases: Array<{ expression: string; options?: Partial<SerialFilterOptions>; matched: boolean }> = [
      { expression: 'len > 0', matched: true },
      { expression: 'text contains "OK"', matched: true },
      { expression: 'hex contains "0d0a"', matched: true },
      { expression: 'len >= 4 and text contains "TEMP"', matched: true },
      { expression: 'not (hex contains "ab")', matched: true },
      { expression: 'level == "error" and message contains "template"', matched: true },
      { expression: 'text contains "ok"', options: { caseSensitive: true }, matched: false },
      { expression: 'text contains "ok"', matched: true },
      { expression: 'text contains "EMP"', options: { wholeWord: true }, matched: true },
    ]

    for (const item of cases) {
      expect(matchSerialFilter(candidate, {
        mode: 'expression',
        expression: item.expression,
        ...item.options,
      }), item.expression).toEqual({ matched: item.matched })
    }
  })

  it('returns structured errors for invalid Wireshark-like expressions', () => {
    const candidate: SerialFilterCandidate = { len: 3, text: 'abc' }

    for (const expression of ['len >> 2', 'text contains', '(len > 1', 'unknown contains "x"']) {
      const result = matchSerialFilter(candidate, { mode: 'expression', expression })
      expect(result.matched, expression).toBe(false)
      expect(result.error, expression).toEqual(expect.any(String))
    }
  })
})
