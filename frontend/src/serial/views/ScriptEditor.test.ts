import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ScriptEditor from './ScriptEditor.vue'
import { completionItemsForModel, resetScriptLanguageState } from './scriptLanguage'

const monacoApi = vi.hoisted(() => {
  const editor = {
    dispose: vi.fn(),
    getValue: vi.fn(() => 'output.bytes(input.bytes())'),
    getModel: vi.fn(() => ({ uri: { toString: () => 'model-uri' } })),
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
        language: 'mocktrue-script',
        automaticLayout: true,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
      })
    )
    expect(wrapper.find('[data-testid="serial-script-editor"]').classes()).toContain('serial-script-editor')

    await wrapper.setProps({ modelValue: 'output.hex("01 02")' })

    expect(monacoApi.editor.setValue).toHaveBeenCalledWith('output.hex("01 02")')
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
      'mocktrue-script',
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
      'mocktrue-script',
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
      'mocktrue-script',
      []
    )
  })
})
