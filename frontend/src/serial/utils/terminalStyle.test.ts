import { describe, expect, it } from 'vitest'
import { serialTerminalFontFamily, serialTerminalStyle } from './terminalStyle'

describe('serialTerminalFontFamily', () => {
  it('quotes font family names that contain spaces for preview styles', () => {
    expect(serialTerminalFontFamily('System Mono')).toBe('"System Mono", monospace')
  })
})

describe('serialTerminalStyle', () => {
  it('quotes font family names that contain spaces', () => {
    expect(serialTerminalStyle('System Mono', 16)).toEqual({
      '--serial-terminal-font-family': '"System Mono", monospace',
      '--serial-terminal-font-size': '16px',
    })
  })

  it('keeps generic monospace unquoted', () => {
    expect(serialTerminalStyle('monospace', 14)).toEqual({
      '--serial-terminal-font-family': 'monospace',
      '--serial-terminal-font-size': '14px',
    })
  })
})
