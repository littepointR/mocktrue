export function serialTerminalStyle(fontFamily: string, fontSize: number): Record<string, string> {
  return {
    '--serial-terminal-font-family': serialTerminalFontFamily(fontFamily),
    '--serial-terminal-font-size': `${fontSize}px`,
  }
}

export function serialTerminalFontFamily(fontFamily: string): string {
  const cssFamily = cssFontFamily(fontFamily)
  return cssFamily === 'monospace' ? cssFamily : `${cssFamily}, monospace`
}

function cssFontFamily(fontFamily: string): string {
  const trimmed = fontFamily.trim()
  if (trimmed === 'monospace') return 'monospace'
  if (/^[a-zA-Z0-9-]+$/.test(trimmed)) return trimmed
  return `"${trimmed.replace(/"/g, '\\"')}"`
}
