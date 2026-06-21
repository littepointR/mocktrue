<script setup lang="ts">
import * as monaco from 'monaco-editor'
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import {
  buildScriptDiagnostics,
  registerScriptModel,
  registerScriptLanguage,
  scriptLanguageId,
  unregisterScriptModel,
} from './scriptLanguage'

const props = defineProps<{
  modelValue: string
  nodeType: string
}>()

const emit = defineEmits<{
  (event: 'script-change', value: string): void
  (event: 'update:modelValue', value: string): void
}>()

type Editor = ReturnType<typeof monaco.editor.create>

const editorEl = ref<HTMLElement | null>(null)
let editor: Editor | null = null
let changeDisposable: { dispose: () => void } | null = null
let internalChange = false
let modelUri = ''

onMounted(() => {
  if (!editorEl.value) return
  registerScriptLanguage(monaco)
  editor = monaco.editor.create(editorEl.value, {
    value: props.modelValue,
    language: scriptLanguageId,
    theme: 'vs-dark',
    automaticLayout: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    fontSize: 12,
  })
  syncModelNodeType()
  changeDisposable = editor.onDidChangeModelContent(() => {
    if (!editor) return
    internalChange = true
    const value = editor.getValue()
    updateMarkers()
    emit('update:modelValue', value)
    emit('script-change', value)
    void nextTick(() => {
      internalChange = false
    })
  })
  updateMarkers()
})

watch(
  () => props.modelValue,
  value => {
    if (!editor || internalChange || editor.getValue() === value) return
    editor.setValue(value)
    updateMarkers()
  }
)

watch(
  () => props.nodeType,
  () => {
    syncModelNodeType()
    updateMarkers()
  }
)

onUnmounted(() => {
  if (modelUri) {
    unregisterScriptModel(modelUri)
    modelUri = ''
  }
  changeDisposable?.dispose()
  changeDisposable = null
  editor?.dispose()
  editor = null
})

function syncModelNodeType() {
  const model = editor?.getModel()
  const uri = model?.uri?.toString() ?? ''
  if (!uri) return
  if (modelUri && modelUri !== uri) {
    unregisterScriptModel(modelUri)
  }
  modelUri = uri
  registerScriptModel(modelUri, props.nodeType)
}

function updateMarkers() {
  const model = editor?.getModel()
  if (!model) return
  monaco.editor.setModelMarkers(
    model,
    scriptLanguageId,
    buildScriptDiagnostics(editor?.getValue() ?? props.modelValue, props.nodeType, monaco)
  )
}
</script>

<template>
  <div
    ref="editorEl"
    class="serial-script-editor"
    data-testid="serial-script-editor"
  />
</template>

<style scoped>
.serial-script-editor {
  width: 100%;
  height: 220px;
  min-height: 160px;
  max-height: 260px;
  overflow: hidden;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  background: var(--app-bg, #1e1e1e);
}
</style>
