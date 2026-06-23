export const scriptLanguageId = 'portweave-script'

export type ScriptNodeType =
  | 'serial.script.transform'
  | 'serial.script.generator'
  | 'serial.script.analyzer'

export interface ScriptModelLike {
  uri?: { toString: () => string }
  getLineContent: (lineNumber: number) => string
  getWordAtPosition?: (position: { lineNumber: number; column: number }) => { word: string; startColumn: number; endColumn: number } | null
}

export interface ScriptDiagnostic {
  startLineNumber: number
  startColumn: number
  endLineNumber: number
  endColumn: number
  message: string
  severity: number
}

interface ScriptSymbol {
  label: string
  hoverWord: string
  insertText: string
  detail: string
  documentation: string
  kind: 'function' | 'variable' | 'field'
  memberOf?: 'input' | 'output' | 'state'
  memberLabel?: string
  memberInsertText?: string
  unavailableFor?: ScriptNodeType[]
}

type CompletionContext =
  | { kind: 'all' }
  | { kind: 'member'; objectName: 'input' | 'output' | 'state' }

type MonacoLike = {
  MarkerSeverity?: {
    Error?: number
    Warning?: number
  }
  languages: {
    CompletionItemKind?: Record<string, unknown>
    CompletionItemInsertTextRule?: Record<string, unknown>
    register?: (language: { id: string }) => unknown
    registerCompletionItemProvider?: (...args: any[]) => unknown
    registerHoverProvider?: (...args: any[]) => unknown
    setMonarchTokensProvider?: (...args: any[]) => unknown
    setLanguageConfiguration?: (...args: any[]) => unknown
  }
}

const scriptKeywords = [
  'break', 'case', 'catch', 'const', 'continue', 'default', 'do', 'else', 'false',
  'finally', 'for', 'function', 'if', 'let', 'new', 'null', 'return', 'switch',
  'throw', 'true', 'try', 'typeof', 'undefined', 'var', 'while',
]

const scriptBuiltins = [
  'input', 'output', 'state', 'field', 'error', 'drop', 'crc16', 'sum8', 'now',
]

const scriptMonarchLanguage = {
  defaultToken: '',
  tokenPostfix: '.portweave-script',
  keywords: scriptKeywords,
  builtins: scriptBuiltins,
  tokenizer: {
    root: [
      [/\/\/.*$/, 'comment'],
      [/\/\*/, 'comment', '@comment'],
      [/"([^"\\]|\\.)*$/, 'string.invalid'],
      [/"/, 'string', '@string_double'],
      [/'([^'\\]|\\.)*$/, 'string.invalid'],
      [/'/, 'string', '@string_single'],
      [/`/, 'string', '@string_backtick'],
      [/\b(?:0[xX][0-9a-fA-F]+|\d+(?:\.\d+)?)\b/, 'number'],
      [/[{}()[\]]/, '@brackets'],
      [/[;,.]/, 'delimiter'],
      [/[+\-*\/%=<>!&|?:]+/, 'operator'],
      [/[a-zA-Z_$][\w$]*/, {
        cases: {
          '@keywords': 'keyword',
          '@builtins': 'predefined',
          '@default': 'identifier',
        },
      }],
      [/\s+/, 'white'],
    ],
    comment: [
      [/[^/*]+/, 'comment'],
      [/\*\//, 'comment', '@pop'],
      [/[/*]/, 'comment'],
    ],
    string_double: [
      [/[^\\"]+/, 'string'],
      [/\\./, 'string.escape'],
      [/"/, 'string', '@pop'],
    ],
    string_single: [
      [/[^\\']+/, 'string'],
      [/\\./, 'string.escape'],
      [/'/, 'string', '@pop'],
    ],
    string_backtick: [
      [/[^\\`]+/, 'string'],
      [/\\./, 'string.escape'],
      [/`/, 'string', '@pop'],
    ],
  },
}

const scriptLanguageConfiguration = {
  comments: {
    lineComment: '//',
    blockComment: ['/*', '*/'],
  },
  brackets: [
    ['{', '}'],
    ['[', ']'],
    ['(', ')'],
  ],
  autoClosingPairs: [
    { open: '{', close: '}' },
    { open: '[', close: ']' },
    { open: '(', close: ')' },
    { open: '"', close: '"' },
    { open: "'", close: "'" },
    { open: '`', close: '`' },
  ],
  surroundingPairs: [
    { open: '{', close: '}' },
    { open: '[', close: ']' },
    { open: '(', close: ')' },
    { open: '"', close: '"' },
    { open: "'", close: "'" },
    { open: '`', close: '`' },
  ],
}

const symbols: ScriptSymbol[] = [
  {
    label: 'input.bytes()',
    hoverWord: 'input.bytes',
    insertText: 'input.bytes()',
    memberOf: 'input',
    memberLabel: 'bytes()',
    memberInsertText: 'bytes()',
    detail: 'input.bytes(): number[]',
    documentation: '返回当前节点接收到的输入字节数组。generator 节点没有 input.* 上下文。',
    kind: 'function',
    unavailableFor: ['serial.script.generator'],
  },
  {
    label: 'input.hex()',
    hoverWord: 'input.hex',
    insertText: 'input.hex()',
    memberOf: 'input',
    memberLabel: 'hex()',
    memberInsertText: 'hex()',
    detail: 'input.hex(): string',
    documentation: '返回当前输入字节的空格分隔 hex 字符串。generator 节点没有 input.* 上下文。',
    kind: 'function',
    unavailableFor: ['serial.script.generator'],
  },
  {
    label: 'input.text(encoding)',
    hoverWord: 'input.text',
    insertText: 'input.text("${1:utf-8}")',
    memberOf: 'input',
    memberLabel: 'text(encoding)',
    memberInsertText: 'text("${1:utf-8}")',
    detail: 'input.text(encoding?: string): string',
    documentation: '按编码把当前输入字节解码为文本。generator 节点没有 input.* 上下文。',
    kind: 'function',
    unavailableFor: ['serial.script.generator'],
  },
  {
    label: 'output.bytes(bytes)',
    hoverWord: 'output.bytes',
    insertText: 'output.bytes($1)',
    memberOf: 'output',
    memberLabel: 'bytes(bytes)',
    memberInsertText: 'bytes($1)',
    detail: 'output.bytes(bytes: number[]): void',
    documentation: '输出字节流到下游节点。analyzer 节点不提供 output.*。',
    kind: 'function',
    unavailableFor: ['serial.script.analyzer'],
  },
  {
    label: 'output.hex(hex)',
    hoverWord: 'output.hex',
    insertText: 'output.hex("$1")',
    memberOf: 'output',
    memberLabel: 'hex(hex)',
    memberInsertText: 'hex("$1")',
    detail: 'output.hex(hex: string): void',
    documentation: '输出空格分隔的 hex 字节流到下游节点。analyzer 节点不提供 output.*。',
    kind: 'function',
    unavailableFor: ['serial.script.analyzer'],
  },
  {
    label: 'output.text(text, encoding)',
    hoverWord: 'output.text',
    insertText: 'output.text("$1", "${2:utf-8}")',
    memberOf: 'output',
    memberLabel: 'text(text, encoding)',
    memberInsertText: 'text("$1", "${2:utf-8}")',
    detail: 'output.text(text: string, encoding?: string): void',
    documentation: '按编码输出文本到下游节点。analyzer 节点不提供 output.*。',
    kind: 'function',
    unavailableFor: ['serial.script.analyzer'],
  },
  {
    label: 'field(name, value)',
    hoverWord: 'field',
    insertText: 'field("$1", $2)',
    detail: 'field(name, value)',
    documentation: '记录一个分析字段，供节点内容区展示。',
    kind: 'function',
  },
  {
    label: 'error(message)',
    hoverWord: 'error',
    insertText: 'error("$1")',
    detail: 'error(message)',
    documentation: '记录脚本错误。转换和生成节点会停止本次输出并标记错误；分析节点会把错误写入帧记录。',
    kind: 'function',
  },
  {
    label: 'drop()',
    hoverWord: 'drop',
    insertText: 'drop()',
    detail: 'drop()',
    documentation: '丢弃当前输入，不向下游输出。',
    kind: 'function',
  },
  {
    label: 'state.get(key)',
    hoverWord: 'state.get',
    insertText: 'state.get("$1")',
    memberOf: 'state',
    memberLabel: 'get(key)',
    memberInsertText: 'get("$1")',
    detail: 'state.get(key: string): unknown',
    documentation: '读取当前脚本节点的受限状态值。',
    kind: 'function',
  },
  {
    label: 'state.set(key, value)',
    hoverWord: 'state.set',
    insertText: 'state.set("$1", $2)',
    memberOf: 'state',
    memberLabel: 'set(key, value)',
    memberInsertText: 'set("$1", $2)',
    detail: 'state.set(key: string, value: unknown): void',
    documentation: '写入当前脚本节点的受限状态值，受 maxStateBytes 限制。',
    kind: 'function',
  },
  {
    label: 'state.delete(key)',
    hoverWord: 'state.delete',
    insertText: 'state.delete("$1")',
    memberOf: 'state',
    memberLabel: 'delete(key)',
    memberInsertText: 'delete("$1")',
    detail: 'state.delete(key: string): void',
    documentation: '删除当前脚本节点的状态值。',
    kind: 'function',
  },
  {
    label: 'crc16(bytes)',
    hoverWord: 'crc16',
    insertText: 'crc16($1)',
    detail: 'crc16(bytes)',
    documentation: '计算 CRC16 校验值。',
    kind: 'function',
  },
  {
    label: 'sum8(bytes)',
    hoverWord: 'sum8',
    insertText: 'sum8($1)',
    detail: 'sum8(bytes)',
    documentation: '计算 8 位累加和。',
    kind: 'function',
  },
  {
    label: 'now()',
    hoverWord: 'now',
    insertText: 'now()',
    detail: 'now()',
    documentation: '返回当前时间戳。',
    kind: 'function',
  },
]

let registered = false
const scriptModelNodeTypes = new Map<string, ScriptNodeType>()

export function isScriptNodeType(type: string): type is ScriptNodeType {
  return type === 'serial.script.transform'
    || type === 'serial.script.generator'
    || type === 'serial.script.analyzer'
}

export function registerScriptModel(modelUri: string, nodeType: string) {
  if (!modelUri) return
  if (isScriptNodeType(nodeType)) {
    scriptModelNodeTypes.set(modelUri, nodeType)
  } else {
    scriptModelNodeTypes.delete(modelUri)
  }
}

export function unregisterScriptModel(modelUri: string) {
  if (!modelUri) return
  scriptModelNodeTypes.delete(modelUri)
}

export function resetScriptLanguageState() {
  registered = false
  scriptModelNodeTypes.clear()
}

export function completionLabelsForScriptNode(nodeType: string): string[] {
  return symbolsForNode(nodeType).map(symbol => symbol.label)
}

export function scriptHoverForWord(nodeType: string, word: string): { contents: Array<{ value: string }> } | null {
  const symbol = symbolsForNode(nodeType).find(item => item.hoverWord === word || item.label === word)
  if (!symbol) return null
  return {
    contents: [
      { value: `**${symbol.label}** - ${symbol.detail}\n\n${symbol.documentation}` },
    ],
  }
}

export function completionItemsForModel(
  monaco: MonacoLike,
  model: ScriptModelLike,
  position?: { lineNumber: number; column: number }
) {
  return completionItemsForNode(monaco, scriptNodeTypeForModel(model), completionContextForModel(model, position))
}

export function scriptHoverForModel(model: ScriptModelLike, position: { lineNumber: number; column: number }) {
  const word = scriptWordAtPosition(model, position)
  if (!word) return null
  return scriptHoverForWord(scriptNodeTypeForModel(model), word)
}

export function buildScriptDiagnostics(script: string, nodeType: string, monaco?: MonacoLike): ScriptDiagnostic[] {
  const severity = monaco?.MarkerSeverity?.Error ?? 8
  const diagnostics: ScriptDiagnostic[] = []
  const source = script.trim()

  if (!source) {
    diagnostics.push(markerFor(script, '脚本不能为空', severity))
    return diagnostics
  }

  if (/\bserial\s*\./.test(script)) {
    diagnostics.push(markerForMatch(script, /\bserial\s*\./, '第一阶段不提供 serial API', severity))
  }

  if (nodeType === 'serial.script.generator' && /\binput\s*\./.test(script)) {
    diagnostics.push(markerForMatch(script, /\binput\s*\./, 'generator 脚本没有 input.* 上下文', severity))
  }

  if (nodeType === 'serial.script.analyzer' && /\boutput\s*\./.test(script)) {
    diagnostics.push(markerForMatch(script, /\boutput\s*\./, 'analyzer 脚本没有 output.* 上下文', severity))
  }

  if (/\binput\s*\(/.test(script)) {
    diagnostics.push(markerForMatch(script, /\binput\s*\(/, '请使用 input.bytes()/input.hex()/input.text(encoding)', severity))
  }

  if (/\boutput\s*\(/.test(script)) {
    diagnostics.push(markerForMatch(script, /\boutput\s*\(/, '请使用 output.bytes()/output.hex()/output.text(text, encoding)', severity))
  }

  return diagnostics
}

export function registerScriptLanguage(monaco: MonacoLike) {
  if (registered) return
  registered = true

  monaco.languages.register?.({ id: scriptLanguageId })
  monaco.languages.setMonarchTokensProvider?.(scriptLanguageId, scriptMonarchLanguage)
  monaco.languages.setLanguageConfiguration?.(scriptLanguageId, scriptLanguageConfiguration)
  monaco.languages.registerCompletionItemProvider?.(scriptLanguageId, {
    triggerCharacters: ['.', '('],
    provideCompletionItems: (model: ScriptModelLike, position: { lineNumber: number; column: number }) => ({
      suggestions: completionItemsForModel(monaco, model, position),
    }),
  })
  monaco.languages.registerHoverProvider?.(scriptLanguageId, {
    provideHover: (model: ScriptModelLike, position: { lineNumber: number; column: number }) => scriptHoverForModel(model, position),
  })
}

export function completionItemsForNode(
  monaco: MonacoLike,
  nodeType: string,
  context: CompletionContext = { kind: 'all' }
) {
  const kinds = monaco.languages.CompletionItemKind ?? {}
  const insertTextRules = monaco.languages.CompletionItemInsertTextRule ?? {}
  return symbolsForNode(nodeType)
    .filter(symbol => context.kind === 'all' || symbol.memberOf === context.objectName)
    .map(symbol => ({
      label: context.kind === 'member' ? (symbol.memberLabel ?? symbol.label) : symbol.label,
      kind: kindForSymbol(symbol, kinds),
      detail: symbol.detail,
      documentation: symbol.documentation,
      insertText: context.kind === 'member' ? (symbol.memberInsertText ?? symbol.insertText) : symbol.insertText,
      insertTextRules: numberKind(insertTextRules.InsertAsSnippet, 4),
    }))
}

function symbolsForNode(nodeType: string): ScriptSymbol[] {
  if (!isScriptNodeType(nodeType)) return symbols
  return symbols.filter(symbol => !symbol.unavailableFor?.includes(nodeType))
}

function scriptNodeTypeForModel(model: ScriptModelLike): ScriptNodeType {
  const uri = model.uri?.toString() ?? ''
  return (scriptModelNodeTypes.get(uri) ?? 'serial.script.transform') as ScriptNodeType
}

function completionContextForModel(
  model: ScriptModelLike,
  position?: { lineNumber: number; column: number }
): CompletionContext {
  if (!position) return { kind: 'all' }
  const line = model.getLineContent(position.lineNumber) ?? ''
  const beforeCursor = line.slice(0, Math.max(0, position.column - 1))
  const memberMatch = /\b(input|output|state)\.$/.exec(beforeCursor)
  if (memberMatch) {
    return { kind: 'member', objectName: memberMatch[1] as 'input' | 'output' | 'state' }
  }
  return { kind: 'all' }
}

function scriptWordAtPosition(model: ScriptModelLike, position: { lineNumber: number; column: number }): string | null {
  const line = model.getLineContent(position.lineNumber) ?? ''
  if (!line) return null

  const index = Math.max(0, Math.min(line.length - 1, position.column - 1))
  for (const token of dottedScriptTokens(line)) {
    if (index >= token.start && index < token.end) {
      return token.value
    }
  }

  const baseWord = model.getWordAtPosition?.(position)
  if (baseWord?.word) {
    return baseWord.word
  }

  const start = expandScriptWordStart(line, index)
  const end = expandScriptWordEnd(line, index + 1)
  const token = line.slice(start, end)
  return token || null
}

function expandScriptWordStart(line: string, start: number): number {
  let next = Math.max(0, start)
  while (next > 0 && isScriptTokenChar(line[next - 1])) {
    next -= 1
  }
  return next
}

function expandScriptWordEnd(line: string, end: number): number {
  let next = Math.min(line.length, end)
  while (next < line.length && isScriptTokenChar(line[next])) {
    next += 1
  }
  return next
}

function isScriptTokenChar(char: string | undefined): boolean {
  return typeof char === 'string' && /[A-Za-z0-9_.]/.test(char)
}

function dottedScriptTokens(line: string): Array<{ start: number; end: number; value: string }> {
  const tokens: Array<{ start: number; end: number; value: string }> = []
  const pattern = /[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+/g
  let match: RegExpExecArray | null
  while ((match = pattern.exec(line))) {
    tokens.push({ start: match.index, end: match.index + match[0].length, value: match[0] })
  }
  return tokens
}

function kindForSymbol(symbol: ScriptSymbol, kinds: Record<string, unknown>): number {
  if (symbol.kind === 'function') return numberKind(kinds.Function, 1)
  if (symbol.kind === 'field') return numberKind(kinds.Field, 3)
  return numberKind(kinds.Variable, 2)
}

function numberKind(value: unknown, fallback: number): number {
  return typeof value === 'number' ? value : fallback
}

function markerFor(script: string, message: string, severity: number): ScriptDiagnostic {
  const lines = script.split('\n')
  const lastLine = Math.max(1, lines.length)
  const lastColumn = Math.max(1, (lines[lines.length - 1] ?? '').length + 1)
  return {
    startLineNumber: 1,
    startColumn: 1,
    endLineNumber: lastLine,
    endColumn: lastColumn,
    message,
    severity,
  }
}

function markerForMatch(script: string, pattern: RegExp, message: string, severity: number): ScriptDiagnostic {
  const match = pattern.exec(script)
  if (!match || match.index < 0) return markerFor(script, message, severity)
  const prefix = script.slice(0, match.index)
  const lines = prefix.split('\n')
  const line = lines.length
  const column = (lines[lines.length - 1] ?? '').length + 1
  const endColumn = column + match[0].length
  return {
    startLineNumber: line,
    startColumn: column,
    endLineNumber: line,
    endColumn,
    message,
    severity,
  }
}
