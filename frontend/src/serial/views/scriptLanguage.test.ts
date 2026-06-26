import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  buildScriptDiagnostics,
  completionLabelsForScriptNode,
  completionItemsForNode,
  completionItemsForModel,
  registerScriptModel,
  registerScriptLanguage,
  resetScriptLanguageState,
  scriptHoverForWord,
  scriptHoverForModel,
  unregisterScriptModel,
} from './scriptLanguage'

describe('scriptLanguage', () => {
  beforeEach(() => {
    resetScriptLanguageState()
  })

  it('filters completion globals by script node type', () => {
    expect(completionLabelsForScriptNode('serial.script.transform')).toEqual(expect.arrayContaining([
      'input.bytes()',
      'input.hex()',
      'input.text(encoding)',
      'output.bytes(bytes)',
      'output.hex(hex)',
      'output.text(text, encoding)',
      'field(name, value)',
      'error(message)',
      'drop()',
      'state.get(key)',
      'state.set(key, value)',
      'state.delete(key)',
      'crc16(bytes)',
      'sum8(bytes)',
      'now()',
    ]))

    expect(completionLabelsForScriptNode('serial.script.generator')).not.toContain('input.bytes()')
    expect(completionLabelsForScriptNode('serial.script.generator')).toContain('output.bytes(bytes)')

    expect(completionLabelsForScriptNode('serial.script.analyzer')).toContain('input.bytes()')
    expect(completionLabelsForScriptNode('serial.script.analyzer')).not.toContain('output.bytes(bytes)')
  })

  it('provides hover content for supported helpers and hides unavailable words', () => {
    expect(scriptHoverForWord('serial.script.transform', 'crc16')?.contents[0].value).toContain('CRC16')
    expect(scriptHoverForWord('serial.script.generator', 'input.bytes')).toBeNull()
    expect(scriptHoverForWord('serial.script.analyzer', 'output.bytes')).toBeNull()
  })

  it('resolves completion and hover from the bound Monaco model node type', () => {
    const model = {
      uri: { toString: () => 'inmemory://script-generator' },
      getLineContent: () => 'output.bytes(input.bytes())',
      getWordAtPosition: () => ({ word: 'bytes', startColumn: 9, endColumn: 14 }),
    }
    registerScriptModel(model.uri.toString(), 'serial.script.generator')

    expect(completionItemsForModel({
      languages: {
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
      },
    } as any, model as any).map(item => item.label)).not.toContain('input.bytes()')
    expect(scriptHoverForModel(model as any, { lineNumber: 1, column: 9 })?.contents[0].value).toContain('output.bytes')
  })

  it('narrows completion items for dotted PortWeave API objects', () => {
    const monaco = {
      languages: {
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
      },
    }
    const model = {
      uri: { toString: () => 'inmemory://script-transform' },
      getLineContent: () => 'input.',
      getWordAtPosition: () => null,
    }
    registerScriptModel(model.uri.toString(), 'serial.script.transform')

    const labels = completionItemsForModel(monaco as any, model as any, { lineNumber: 1, column: 7 }).map(item => item.label)

    expect(labels).toEqual(['bytes()', 'hex()', 'text(encoding)'])
  })

  it('does not suggest unavailable dotted APIs for node types', () => {
    const monaco = {
      languages: {
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
      },
    }
    const generatorModel = {
      uri: { toString: () => 'inmemory://script-generator' },
      getLineContent: () => 'input.',
      getWordAtPosition: () => null,
    }
    registerScriptModel(generatorModel.uri.toString(), 'serial.script.generator')

    expect(completionItemsForModel(monaco as any, generatorModel as any, { lineNumber: 1, column: 7 })).toEqual([])

    const analyzerModel = {
      uri: { toString: () => 'inmemory://script-analyzer' },
      getLineContent: () => 'output.',
      getWordAtPosition: () => null,
    }
    registerScriptModel(analyzerModel.uri.toString(), 'serial.script.analyzer')

    expect(completionItemsForModel(monaco as any, analyzerModel as any, { lineNumber: 1, column: 8 })).toEqual([])
  })

  it('reports basic diagnostics without exposing the frozen-out serial API', () => {
    expect(buildScriptDiagnostics('', 'serial.script.transform').map(item => item.message)).toEqual([
      '脚本不能为空',
    ])

    expect(buildScriptDiagnostics('serial.write(input.bytes())\noutput.bytes(input.bytes())', 'serial.script.transform').map(item => item.message)).toEqual([
      '第一阶段不提供 serial API',
    ])

    expect(buildScriptDiagnostics('output.bytes(input.bytes())', 'serial.script.generator').map(item => item.message)).toContain(
      'generator 脚本没有 input.* 上下文'
    )
    expect(buildScriptDiagnostics('output.bytes(input.bytes())', 'serial.script.analyzer').map(item => item.message)).toContain(
      'analyzer 脚本没有 output.* 上下文'
    )
    expect(buildScriptDiagnostics('output(input)', 'serial.script.transform').map(item => item.message)).toContain(
      '请使用 output.bytes()/output.hex()/output.text(text, encoding)'
    )
  })

  it('registers Monaco completion and hover providers once', () => {
    const monaco = {
      languages: {
        register: vi.fn(),
        registerCompletionItemProvider: vi.fn(),
        registerHoverProvider: vi.fn(),
        setMonarchTokensProvider: vi.fn(),
        setLanguageConfiguration: vi.fn(),
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
      },
    }

    registerScriptLanguage(monaco as any)
    registerScriptLanguage(monaco as any)

    expect(monaco.languages.register).toHaveBeenCalledTimes(1)
    expect(monaco.languages.registerCompletionItemProvider).toHaveBeenCalledTimes(1)
    expect(monaco.languages.registerHoverProvider).toHaveBeenCalledTimes(1)
  })

  it('registers completion trigger characters for dot member access', () => {
    const monaco = {
      languages: {
        register: vi.fn(),
        registerCompletionItemProvider: vi.fn(),
        registerHoverProvider: vi.fn(),
        setMonarchTokensProvider: vi.fn(),
        setLanguageConfiguration: vi.fn(),
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
      },
    }

    registerScriptLanguage(monaco as any)

    expect(monaco.languages.registerCompletionItemProvider).toHaveBeenCalledWith(
      'portweave-script',
      expect.objectContaining({
        triggerCharacters: ['.', '('],
        provideCompletionItems: expect.any(Function),
      })
    )
  })

  it('registers Monaco syntax highlighting and language configuration once', () => {
    const monaco = {
      languages: {
        register: vi.fn(),
        registerCompletionItemProvider: vi.fn(),
        registerHoverProvider: vi.fn(),
        setMonarchTokensProvider: vi.fn(),
        setLanguageConfiguration: vi.fn(),
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
      },
    }

    registerScriptLanguage(monaco as any)
    registerScriptLanguage(monaco as any)

    expect(monaco.languages.setMonarchTokensProvider).toHaveBeenCalledTimes(1)
    expect(monaco.languages.setMonarchTokensProvider).toHaveBeenCalledWith(
      'portweave-script',
      expect.objectContaining({
        tokenizer: expect.objectContaining({
          root: expect.any(Array),
        }),
      })
    )
    expect(monaco.languages.setLanguageConfiguration).toHaveBeenCalledTimes(1)
    expect(monaco.languages.setLanguageConfiguration).toHaveBeenCalledWith(
      'portweave-script',
      expect.objectContaining({
        comments: expect.objectContaining({ lineComment: '//' }),
        brackets: expect.arrayContaining([['{', '}'], ['[', ']'], ['(', ')']]),
        autoClosingPairs: expect.arrayContaining([
          expect.objectContaining({ open: '"', close: '"' }),
          expect.objectContaining({ open: '(', close: ')' }),
        ]),
      })
    )
  })

  it('falls back from dotted tokens to Monaco words and expanded identifiers', () => {
    const monaco = {
      languages: {
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
        CompletionItemInsertTextRule: { InsertAsSnippet: 99 },
      },
    }
    const untypedModel = {
      uri: { toString: () => 'inmemory://unknown-script' },
      getLineContent: () => 'state.',
      getWordAtPosition: () => null,
    }

    expect(completionItemsForModel(monaco as any, untypedModel as any, { lineNumber: 1, column: 7 })
      .map(item => [item.label, item.insertTextRules])).toEqual([
      ['get(key)', 99],
      ['set(key, value)', 99],
      ['delete(key)', 99],
    ])

    const wordModel = {
      getLineContent: () => 'const value = crc16(bytes)',
      getWordAtPosition: () => ({ word: 'crc16', startColumn: 15, endColumn: 20 }),
    }
    expect(scriptHoverForModel(wordModel as any, { lineNumber: 1, column: 15 })?.contents[0].value).toContain('CRC16')

    const expandedModel = {
      getLineContent: () => '  output.text',
      getWordAtPosition: () => null,
    }
    expect(scriptHoverForModel(expandedModel as any, { lineNumber: 1, column: 12 })?.contents[0].value).toContain('output.text')

    const emptyModel = {
      getLineContent: () => '',
      getWordAtPosition: () => null,
    }
    expect(scriptHoverForModel(emptyModel as any, { lineNumber: 1, column: 1 })).toBeNull()
  })

  it('unregisters model bindings and reports legacy input call diagnostics', () => {
    const model = {
      uri: { toString: () => 'inmemory://script-generator' },
      getLineContent: () => 'input.',
      getWordAtPosition: () => null,
    }
    registerScriptModel(model.uri.toString(), 'serial.script.generator')
    expect(completionItemsForModel({ languages: {} } as any, model as any, { lineNumber: 1, column: 7 })).toEqual([])

    unregisterScriptModel(model.uri.toString())

    expect(completionItemsForModel({ languages: {} } as any, model as any, { lineNumber: 1, column: 7 })
      .map(item => item.label)).toEqual(['bytes()', 'hex()', 'text(encoding)'])
    expect(buildScriptDiagnostics('input()', 'serial.script.transform').map(item => item.message)).toContain(
      '请使用 input.bytes()/input.hex()/input.text(encoding)'
    )
  })

  it('ignores blank or unsupported model registrations and falls back to transform completions', () => {
    const model = {
      uri: { toString: () => 'inmemory://unsupported-script' },
      getLineContent: () => 'output.',
      getWordAtPosition: () => null,
    }

    registerScriptModel('', 'serial.script.generator')
    registerScriptModel(model.uri.toString(), 'serial.script.generator')
    expect(completionItemsForModel({ languages: {} } as any, model as any, { lineNumber: 1, column: 8 })).toEqual([
      expect.objectContaining({ label: 'bytes(bytes)' }),
      expect.objectContaining({ label: 'hex(hex)' }),
      expect.objectContaining({ label: 'text(text, encoding)' }),
    ])

    registerScriptModel(model.uri.toString(), 'serial.script.unknown')
    unregisterScriptModel('')

    expect(completionItemsForModel({ languages: {} } as any, model as any, { lineNumber: 1, column: 8 })
      .map(item => item.label)).toEqual(['bytes(bytes)', 'hex(hex)', 'text(text, encoding)'])
  })

  it('uses fallback symbol kinds and precise diagnostic ranges for multiline scripts', () => {
    const items = completionItemsForModel({ languages: {} } as any, {
      getLineContent: () => '',
    } as any)

    expect(items.find(item => item.label === 'input.bytes()')).toEqual(expect.objectContaining({ kind: 1, insertTextRules: 4 }))

    const diagnostics = buildScriptDiagnostics('const x = 1\n  serial.write()\noutput(input)', 'serial.script.transform', {
      MarkerSeverity: { Error: 99 },
      languages: {},
    } as any)

    expect(diagnostics).toEqual(expect.arrayContaining([
      expect.objectContaining({ message: '第一阶段不提供 serial API', severity: 99, startLineNumber: 2, startColumn: 3 }),
      expect.objectContaining({ message: '请使用 output.bytes()/output.hex()/output.text(text, encoding)', severity: 99, startLineNumber: 3 }),
    ]))
  })

  it('returns null hover when cursor expansion finds no token', () => {
    const punctuationModel = {
      getLineContent: () => '  () ',
      getWordAtPosition: () => null,
    }

    expect(scriptHoverForWord('serial.script.transform', 'missing')).toBeNull()
    expect(scriptHoverForModel(punctuationModel as any, { lineNumber: 1, column: 3 })).toBeNull()
  })

  it('uses fallback completion contexts for unknown node types and missing model lines', () => {
    const monaco = {
      languages: {
        CompletionItemKind: {
          Function: 11,
          Variable: 22,
          Field: 33,
        },
      },
    }

    const unknownNodeItems = completionItemsForNode(monaco as any, 'serial.script.unknown')
    expect(unknownNodeItems.map(item => item.label)).toEqual(expect.arrayContaining([
      'input.bytes()',
      'output.bytes(bytes)',
    ]))
    expect(unknownNodeItems.find(item => item.label === 'field(name, value)')?.kind).toBe(11)

    const missingLineModel = {
      getLineContent: () => undefined,
      getWordAtPosition: () => null,
    }
    expect(completionItemsForModel(monaco as any, missingLineModel as any, { lineNumber: 1, column: 1 })
      .map(item => item.label)).toContain('input.bytes()')
    expect(scriptHoverForModel(missingLineModel as any, { lineNumber: 1, column: 1 })).toBeNull()

    const laterWordModel = {
      getLineContent: () => 'output.text crc16',
      getWordAtPosition: () => ({ word: 'crc16', startColumn: 13, endColumn: 18 }),
    }
    expect(scriptHoverForModel(laterWordModel as any, { lineNumber: 1, column: 14 })?.contents[0].value).toContain('CRC16')
  })
})
