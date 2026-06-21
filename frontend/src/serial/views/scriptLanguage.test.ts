import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  buildScriptDiagnostics,
  completionLabelsForScriptNode,
  completionItemsForModel,
  registerScriptModel,
  registerScriptLanguage,
  resetScriptLanguageState,
  scriptHoverForWord,
  scriptHoverForModel,
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
})
