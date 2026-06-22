import { matchSerialFilter, type SerialFilterCandidate, type SerialFilterMode } from './serialFilter'

export type SerialOperationLogLevel = 'debug' | 'info' | 'warn' | 'error'
export type SerialOperationLogLevelFilter = 'all' | SerialOperationLogLevel

export interface SerialOperationLogEntry {
  id: string
  timestamp: string
  level: SerialOperationLogLevel
  source: string
  action: string
  category: string
  message: string
  details?: string
  graphId: string
  nodeId?: string
  direction?: string
  payloadText?: string
  payloadHex?: string
  byteLength?: number
}

export type SerialOperationLogEntryPatch = Partial<Omit<SerialOperationLogEntry, 'id' | 'timestamp' | 'graphId'>> & {
  id?: string
  timestamp?: string
  graphId?: string
}

export interface SerialOperationLogFilter {
  level: SerialOperationLogLevelFilter
  mode: SerialFilterMode
  expression: string
  caseSensitive: boolean
  wholeWord: boolean
}

export interface SerialOperationLogFilterResult {
  entries: SerialOperationLogEntry[]
  error: string | null
}

export const serialOperationLogLimit = 2000

const serialOperationLogLevels: SerialOperationLogLevel[] = ['debug', 'info', 'warn', 'error']
const serialOperationLogLevelFilters: SerialOperationLogLevelFilter[] = ['all', ...serialOperationLogLevels]
const serialFilterModes: SerialFilterMode[] = ['plain', 'regex', 'expression']

export function defaultSerialOperationLogFilter(): SerialOperationLogFilter {
  return {
    level: 'all',
    mode: 'plain',
    expression: '',
    caseSensitive: false,
    wholeWord: false,
  }
}

export function normalizeSerialOperationLogFilter(
  patch: Partial<SerialOperationLogFilter>,
  current: SerialOperationLogFilter = defaultSerialOperationLogFilter()
): SerialOperationLogFilter {
  return {
    level: normalizeOperationLogLevelFilter(patch.level ?? current.level),
    mode: normalizeFilterMode(patch.mode ?? current.mode),
    expression: patch.expression ?? current.expression,
    caseSensitive: patch.caseSensitive ?? current.caseSensitive,
    wholeWord: patch.wholeWord ?? current.wholeWord,
  }
}

export function createSerialOperationLogEntry(
  graphId: string,
  sequence: number,
  patch: SerialOperationLogEntryPatch
): SerialOperationLogEntry {
  return {
    id: patch.id ?? `local:${graphId}:${sequence}`,
    timestamp: patch.timestamp ?? new Date().toISOString(),
    level: normalizeOperationLogLevel(patch.level),
    source: patch.source ?? '',
    action: patch.action ?? '',
    category: patch.category ?? '',
    message: patch.message ?? '',
    details: patch.details,
    graphId: patch.graphId ?? graphId,
    nodeId: patch.nodeId,
    direction: patch.direction,
    payloadText: patch.payloadText,
    payloadHex: patch.payloadHex,
    byteLength: patch.byteLength,
  }
}

export function capSerialOperationLogs(
  entries: SerialOperationLogEntry[],
  limit = serialOperationLogLimit
): SerialOperationLogEntry[] {
  if (entries.length <= limit) return entries
  return entries.slice(entries.length - limit)
}

export function filterSerialOperationLogs(
  entries: SerialOperationLogEntry[],
  filter: SerialOperationLogFilter
): SerialOperationLogFilterResult {
  const expression = filter.expression.trim()
  const levelFiltered = entries.filter(entry => filter.level === 'all' || entry.level === filter.level)
  if (expression === '') return { entries: levelFiltered, error: null }

  const probe = matchSerialFilter({}, {
    mode: filter.mode,
    expression,
    caseSensitive: filter.caseSensitive,
    wholeWord: filter.wholeWord,
  })
  if (probe.error) return { entries: [], error: probe.error }

  const matched: SerialOperationLogEntry[] = []
  for (const entry of levelFiltered) {
    const result = matchSerialFilter(operationLogCandidate(entry), {
      mode: filter.mode,
      expression,
      caseSensitive: filter.caseSensitive,
      wholeWord: filter.wholeWord,
    })
    if (result.error) return { entries: [], error: result.error }
    if (result.matched) matched.push(entry)
  }
  return { entries: matched, error: null }
}

function operationLogCandidate(entry: SerialOperationLogEntry): SerialFilterCandidate {
  const mergedText = [entry.payloadText, entry.message, entry.details].filter(Boolean).join(' ')
  return {
    len: entry.byteLength,
    byteLength: entry.byteLength,
    byteCount: entry.byteLength,
    text: mergedText,
    hex: entry.payloadHex,
    message: entry.message,
    level: entry.level,
    source: entry.source,
    graphId: entry.graphId,
    nodeId: entry.nodeId,
    direction: entry.direction,
    payloadText: entry.payloadText,
    payloadHex: entry.payloadHex,
    details: entry.details,
    action: entry.action,
    category: entry.category,
  }
}

function normalizeOperationLogLevel(value: unknown): SerialOperationLogLevel {
  return serialOperationLogLevels.includes(value as SerialOperationLogLevel)
    ? value as SerialOperationLogLevel
    : 'info'
}

function normalizeOperationLogLevelFilter(value: unknown): SerialOperationLogLevelFilter {
  return serialOperationLogLevelFilters.includes(value as SerialOperationLogLevelFilter)
    ? value as SerialOperationLogLevelFilter
    : 'all'
}

function normalizeFilterMode(value: unknown): SerialFilterMode {
  return serialFilterModes.includes(value as SerialFilterMode)
    ? value as SerialFilterMode
    : 'plain'
}
