import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ScriptEditor from './ScriptEditor.vue'
import { completionItemsForModel, resetScriptLanguageState } from './scriptLanguage'

const monacoApi = vi.hoisted(() => {
  const editor = {
    dispose: vi.fn(),
    getValue: vi.fn(() => 'output.bytes(input.bytes())'),
    getModel: vi.fn((): any => ({ uri: { toString: () => 'model-uri' } })),
    layout: vi.fn(),
    onDidChangeModelContent: vi.fn((callback: () => void) => {
      editor.changeCallback = callback
      return { dispose: vi.fn() }
    }),
    setValue: vi.fn(),
    updateOptions: vi.fn(),
    changeCallback: null as null | (() => void),
  }
  return {
    editor,
    api: {
      MarkerSeverity: { Error: 8, Warning: 4 },
      editor: {
        create: vi.fn(() => editor),
        setModelMarkers: vi.fn(),
      },
      languages: {
        CompletionItemKind: {
          Function: 1,
          Variable: 2,
          Field: 3,
        },
      },
    },
  }
})

vi.mock('monaco-editor', () => monacoApi.api)

describe('ScriptEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetScriptLanguageState()
    monacoApi.editor.changeCallback = null
    monacoApi.editor.getModel.mockReturnValue({ uri: { toString: () => 'model-uri' } } as any)
    monacoApi.editor.getValue.mockReturnValue('output.bytes(input.bytes())')
  })

  it('creates a fixed-layout Monaco editor and syncs script changes', async () => {
    const wrapper = mount(ScriptEditor, {
      props: {
        nodeType: 'serial.script.transform',
        modelValue: 'output.bytes(input.bytes())',
      },
    })

    expect(monacoApi.api.editor.create).toHaveBeenCalledWith(
      expect.any(HTMLElement),
      expect.objectContaining({
        value: 'output.bytes(input.bytes())',
        language: 'portweave-script',
        automaticLayout: true,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        quickSuggestions: { other: true, comments: false, strings: false },
        suggestOnTriggerCharacters: true,
        tabCompletion: 'on',
      })
    )
    expect(wrapper.find('[data-testid="serial-script-editor"]').classes()).toContain('serial-script-editor')

    await wrapper.setProps({ modelValue: 'output.hex("01 02")' })

    expect(monacoApi.editor.setValue).toHaveBeenCalledWith('output.hex("01 02")')
    monacoApi.editor.setValue.mockClear()
    monacoApi.editor.getValue.mockReturnValue('output.hex("03")')
    await wrapper.setProps({ modelValue: 'output.hex("03")' })
    expect(monacoApi.editor.setValue).not.toHaveBeenCalled()
    wrapper.unmount()
    expect(monacoApi.editor.dispose).toHaveBeenCalled()
  })

  it('marks unavailable node APIs as diagnostics', () => {
    monacoApi.editor.getValue.mockReturnValue('output.bytes(input.bytes())')

    mount(ScriptEditor, {
      props: {
        nodeType: 'serial.script.generator',
        modelValue: 'output.bytes(input.bytes())',
      },
    })

    expect(monacoApi.api.editor.setModelMarkers).toHaveBeenCalledWith(
      expect.anything(),
      'portweave-script',
      [expect.objectContaining({ message: 'generator 脚本没有 input.* 上下文' })]
    )
  })

  it('binds the Monaco model to the node type for dynamic completion', () => {
    mount(ScriptEditor, {
      props: {
        nodeType: 'serial.script.generator',
        modelValue: 'output.bytes([1])',
      },
    })

    const labels = completionItemsForModel(monacoApi.api as any, monacoApi.editor.getModel() as any).map(item => item.label)

    expect(labels).not.toContain('input.bytes()')
    expect(labels).toContain('output.bytes(bytes)')
  })

  it('rebinds changed Monaco model URIs when the script node type changes', async () => {
    const generatorModel = {
      uri: { toString: () => 'model-generator' },
      getLineContent: () => 'input.',
      getWordAtPosition: () => null,
    }
    const analyzerModel = {
      uri: { toString: () => 'model-analyzer' },
      getLineContent: () => 'output.',
      getWordAtPosition: () => null,
    }
    monacoApi.editor.getModel.mockReturnValue(generatorModel as any)
    const wrapper = mount(ScriptEditor, {
      props: {
        nodeType: 'serial.script.generator',
        modelValue: 'output.bytes([1])',
      },
    })

    expect(completionItemsForModel(monacoApi.api as any, generatorModel as any, { lineNumber: 1, column: 7 })).toEqual([])

    monacoApi.editor.getModel.mockReturnValue(analyzerModel as any)
    await wrapper.setProps({ nodeType: 'serial.script.analyzer' })

    expect(completionItemsForModel(monacoApi.api as any, generatorModel as any, { lineNumber: 1, column: 7 })
      .map(item => item.label)).toEqual(['bytes()', 'hex()', 'text(encoding)'])
    expect(completionItemsForModel(monacoApi.api as any, analyzerModel as any, { lineNumber: 1, column: 8 })).toEqual([])
  })

  it('skips marker updates when Monaco has no model and ignores stale change callbacks after unmount', () => {
    monacoApi.editor.getModel.mockReturnValue(null as any)
    const wrapper = mount(ScriptEditor, {
      props: {
        nodeType: 'serial.script.transform',
        modelValue: 'output.bytes(input.bytes())',
      },
    })

    expect(monacoApi.api.editor.setModelMarkers).not.toHaveBeenCalled()
    wrapper.unmount()

    expect(() => monacoApi.editor.changeCallback?.()).not.toThrow()
    expect(monacoApi.api.editor.setModelMarkers).not.toHaveBeenCalled()
  })

  it('refreshes diagnostics immediately when the user edits', () => {
    monacoApi.editor.getValue.mockReturnValue('output.bytes([1])')
    mount(ScriptEditor, {
      props: {
        nodeType: 'serial.script.generator',
        modelValue: 'output.bytes([1])',
      },
    })
    monacoApi.api.editor.setModelMarkers.mockClear()

    monacoApi.editor.getValue.mockReturnValue('output.bytes(input.bytes())')
    monacoApi.editor.changeCallback?.()

    expect(monacoApi.api.editor.setModelMarkers).toHaveBeenCalledWith(
      expect.anything(),
      'portweave-script',
      [expect.objectContaining({ message: 'generator 脚本没有 input.* 上下文' })]
    )
  })

  it('does not require return statements for function-style scripts', () => {
    monacoApi.editor.getValue.mockReturnValue('const value = now()\noutput.bytes([value & 0xff])')

    mount(ScriptEditor, {
      props: {
        nodeType: 'serial.script.generator',
        modelValue: 'const value = now()\noutput.bytes([value & 0xff])',
      },
    })

    expect(monacoApi.api.editor.setModelMarkers).toHaveBeenCalledWith(
      expect.anything(),
      'portweave-script',
      []
    )
  })
})
