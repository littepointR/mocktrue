export type SerialFilterMode = 'plain' | 'regex' | 'expression'

export interface SerialFilterOptions {
  mode?: SerialFilterMode | string
  expression?: string
  caseSensitive?: boolean
  wholeWord?: boolean
}

export interface SerialFilterCandidate {
  len?: number
  byteLength?: number
  byteCount?: number
  text?: string
  hex?: string
  message?: string
  level?: string
  source?: string
  graphId?: string
  nodeId?: string
  nodeType?: string
  direction?: string
  payloadText?: string
  payloadHex?: string
  details?: string
  action?: string
  category?: string
}

export interface SerialFilterMatchResult {
  matched: boolean
  error?: string
}

interface Token {
  kind: 'ident' | 'number' | 'string' | 'operator' | 'lparen' | 'rparen' | 'eof'
  value: string
}

export function matchSerialFilter(candidate: SerialFilterCandidate, options: SerialFilterOptions = {}): SerialFilterMatchResult {
  const expression = (options.expression ?? '').trim()
  if (expression === '') return { matched: true }

  const mode = String(options.mode ?? 'plain').trim().toLowerCase()
  try {
    if (mode === 'plain' || mode === '') {
      return { matched: matchPlain(candidate, expression, options) }
    }
    if (mode === 'regex') {
      return { matched: matchRegex(candidate, expression, options) }
    }
    if (mode === 'expression') {
      return { matched: matchExpression(candidate, expression, options) }
    }
    return { matched: false, error: `unknown filter mode: ${options.mode}` }
  } catch (error) {
    return { matched: false, error: error instanceof Error ? error.message : String(error) }
  }
}

function matchPlain(candidate: SerialFilterCandidate, expression: string, options: SerialFilterOptions): boolean {
  let haystack = candidateHaystack(candidate)
  let needle = expression
  if (!options.caseSensitive) {
    haystack = haystack.toLowerCase()
    needle = needle.toLowerCase()
  }
  if (!options.wholeWord) {
    return haystack.includes(needle)
  }
  return new RegExp(`\\b${escapeRegExp(needle)}\\b`).test(haystack)
}

function matchRegex(candidate: SerialFilterCandidate, expression: string, options: SerialFilterOptions): boolean {
  let pattern = expression
  if (options.wholeWord) {
    pattern = `\\b(?:${pattern})\\b`
  }
  let regex: RegExp
  try {
    regex = new RegExp(pattern, options.caseSensitive ? '' : 'i')
  } catch (error) {
    throw new Error(`invalid regex: ${error instanceof Error ? error.message : String(error)}`)
  }
  return regex.test(candidateHaystack(candidate))
}

function matchExpression(candidate: SerialFilterCandidate, expression: string, options: SerialFilterOptions): boolean {
  const parser = new ExpressionParser(tokenizeExpression(expression), candidate, options)
  return parser.parse()
}

function candidateHaystack(candidate: SerialFilterCandidate): string {
  return [
    candidate.text,
    candidate.hex,
    candidate.message,
    candidate.level,
    candidate.source,
    candidate.graphId,
    candidate.nodeId,
    candidate.nodeType,
    candidate.direction,
    candidate.payloadText,
    candidate.payloadHex,
    candidate.details,
    candidate.action,
    candidate.category,
    String(candidateLength(candidate)),
  ].filter(value => value !== undefined && value !== null).join('\n')
}

function candidateLength(candidate: SerialFilterCandidate): number {
  if (typeof candidate.len === 'number') return candidate.len
  if (typeof candidate.byteLength === 'number') return candidate.byteLength
  if (typeof candidate.byteCount === 'number') return candidate.byteCount
  return 0
}

function numericField(candidate: SerialFilterCandidate, field: string): number | null {
  switch (field.toLowerCase()) {
    case 'len':
    case 'bytelength':
    case 'bytecount':
      return candidateLength(candidate)
    default:
      return null
  }
}

function stringField(candidate: SerialFilterCandidate, field: string): string | null {
  switch (field.toLowerCase()) {
    case 'text':
      return candidate.text ?? candidate.payloadText ?? ''
    case 'hex':
      return candidate.hex ?? candidate.payloadHex ?? ''
    case 'message':
      return candidate.message ?? ''
    case 'level':
      return candidate.level ?? ''
    case 'source':
      return candidate.source ?? ''
    case 'graphid':
      return candidate.graphId ?? ''
    case 'nodeid':
      return candidate.nodeId ?? ''
    case 'nodetype':
      return candidate.nodeType ?? ''
    case 'direction':
      return candidate.direction ?? ''
    case 'payloadtext':
      return candidate.payloadText ?? ''
    case 'payloadhex':
      return candidate.payloadHex ?? ''
    case 'details':
      return candidate.details ?? ''
    case 'action':
      return candidate.action ?? ''
    case 'category':
      return candidate.category ?? ''
    default:
      return null
  }
}

function tokenizeExpression(input: string): Token[] {
  const tokens: Token[] = []
  let index = 0
  while (index < input.length) {
    const char = input[index]
    if (/\s/.test(char)) {
      index += 1
      continue
    }
    if (char === '(') {
      tokens.push({ kind: 'lparen', value: char })
      index += 1
      continue
    }
    if (char === ')') {
      tokens.push({ kind: 'rparen', value: char })
      index += 1
      continue
    }
    if (char === '"') {
      const scanned = scanString(input, index)
      tokens.push({ kind: 'string', value: scanned.value })
      index = scanned.next
      continue
    }
    if (/[0-9]/.test(char)) {
      const start = index
      while (index < input.length && /[0-9]/.test(input[index])) index += 1
      tokens.push({ kind: 'number', value: input.slice(start, index) })
      continue
    }
    if (/[A-Za-z_]/.test(char)) {
      const start = index
      while (index < input.length && /[A-Za-z0-9_]/.test(input[index])) index += 1
      tokens.push({ kind: 'ident', value: input.slice(start, index) })
      continue
    }
    if ('=!<>'.includes(char)) {
      const two = input.slice(index, index + 2)
      if (['==', '!=', '>=', '<='].includes(two)) {
        tokens.push({ kind: 'operator', value: two })
        index += 2
        continue
      }
      if (char === '>' || char === '<') {
        tokens.push({ kind: 'operator', value: char })
        index += 1
        continue
      }
    }
    throw new Error(`unexpected token ${JSON.stringify(char)}`)
  }
  tokens.push({ kind: 'eof', value: '' })
  return tokens
}

function scanString(input: string, start: number): { value: string; next: number } {
  let value = ''
  for (let index = start + 1; index < input.length; index += 1) {
    const char = input[index]
    if (char === '"') {
      return { value, next: index + 1 }
    }
    if (char === '\\') {
      if (index + 1 >= input.length) throw new Error('unterminated string literal')
      index += 1
      const escaped = input[index]
      if (escaped === 'n') value += '\n'
      else if (escaped === 'r') value += '\r'
      else if (escaped === 't') value += '\t'
      else value += escaped
      continue
    }
    value += char
  }
  throw new Error('unterminated string literal')
}

class ExpressionParser {
  private position = 0

  constructor(
    private readonly tokens: Token[],
    private readonly candidate: SerialFilterCandidate,
    private readonly options: SerialFilterOptions
  ) {}

  parse(): boolean {
    const matched = this.parseOr()
    if (this.peek().kind !== 'eof') {
      throw new Error(`unexpected token ${JSON.stringify(this.peek().value)}`)
    }
    return matched
  }

  private parseOr(): boolean {
    let left = this.parseAnd()
    while (this.matchKeyword('or')) {
      const right = this.parseAnd()
      left = left || right
    }
    return left
  }

  private parseAnd(): boolean {
    let left = this.parseUnary()
    while (this.matchKeyword('and')) {
      const right = this.parseUnary()
      left = left && right
    }
    return left
  }

  private parseUnary(): boolean {
    if (this.matchKeyword('not')) {
      return !this.parseUnary()
    }
    return this.parsePrimary()
  }

  private parsePrimary(): boolean {
    if (this.matchKind('lparen')) {
      const matched = this.parseOr()
      if (!this.matchKind('rparen')) {
        throw new Error('expected closing parenthesis')
      }
      return matched
    }
    return this.parseComparison()
  }

  private parseComparison(): boolean {
    const fieldToken = this.peek()
    if (fieldToken.kind !== 'ident') {
      throw new Error('expected filter field')
    }
    this.position += 1
    const field = fieldToken.value

    const numberValue = numericField(this.candidate, field)
    if (numberValue !== null) {
      const operator = this.peek()
      if (operator.kind !== 'operator' || !isNumericOperator(operator.value)) {
        throw new Error(`expected numeric comparison after ${field}`)
      }
      this.position += 1
      const numberToken = this.peek()
      if (numberToken.kind !== 'number') {
        throw new Error(`expected numeric literal after ${operator.value}`)
      }
      this.position += 1
      return compareNumber(numberValue, operator.value, Number(numberToken.value))
    }

    const stringValue = stringField(this.candidate, field)
    if (stringValue === null) {
      throw new Error(`unknown filter field: ${field}`)
    }
    if (this.matchKeyword('contains')) {
      return compareString(field, stringValue, 'contains', this.expectStringLiteral('after contains'), this.options)
    }
    const operator = this.peek()
    if (operator.kind !== 'operator' || (operator.value !== '==' && operator.value !== '!=')) {
      throw new Error(`expected string comparison after ${field}`)
    }
    this.position += 1
    return compareString(field, stringValue, operator.value, this.expectStringLiteral(`after ${operator.value}`), this.options)
  }

  private expectStringLiteral(context: string): string {
    const token = this.peek()
    if (token.kind !== 'string') {
      throw new Error(`expected string literal ${context}`)
    }
    this.position += 1
    return token.value
  }

  private peek(): Token {
    return this.tokens[this.position] ?? { kind: 'eof', value: '' }
  }

  private matchKind(kind: Token['kind']): boolean {
    if (this.peek().kind !== kind) return false
    this.position += 1
    return true
  }

  private matchKeyword(keyword: string): boolean {
    const token = this.peek()
    if (token.kind !== 'ident' || token.value.toLowerCase() !== keyword) return false
    this.position += 1
    return true
  }
}

function isNumericOperator(operator: string): boolean {
  return ['==', '!=', '>', '>=', '<', '<='].includes(operator)
}

function compareNumber(left: number, operator: string, right: number): boolean {
  switch (operator) {
    case '==':
      return left === right
    case '!=':
      return left !== right
    case '>':
      return left > right
    case '>=':
      return left >= right
    case '<':
      return left < right
    case '<=':
      return left <= right
    default:
      return false
  }
}

function compareString(field: string, left: string, operator: string, right: string, options: SerialFilterOptions): boolean {
  if (['hex', 'payloadhex'].includes(field.toLowerCase())) {
    left = normalizeHex(left)
    right = normalizeHex(right)
  } else if (!options.caseSensitive) {
    left = left.toLowerCase()
    right = right.toLowerCase()
  }

  if (operator === 'contains') return left.includes(right)
  if (operator === '==') return left === right
  if (operator === '!=') return left !== right
  return false
}

function normalizeHex(value: string): string {
  return value.replace(/\s+/g, '').toLowerCase()
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
