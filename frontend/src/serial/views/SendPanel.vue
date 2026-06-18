<script setup lang="ts">
import { ref } from 'vue'
import { NInput, NButton, NSelect, NSpace } from 'naive-ui'
import { Send } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'

const props = defineProps<{
  handleId: string
}>()

const sendData = ref('')
const sendMode = ref<'ascii' | 'hex'>('ascii')
const error = ref<string | null>(null)

async function handleSend() {
  if (!props.handleId || !sendData.value) return
  try {
    await Send({
      PortID: props.handleId,
      Content: sendData.value,
      Mode: sendMode.value,
    })
    sendData.value = ''
    error.value = null
  } catch (e: any) {
    error.value = e?.message ?? 'Send failed'
  }
}
</script>

<template>
  <div class="send-panel">
    <NSpace align="center">
      <NSelect
        v-model:value="sendMode"
        :options="[
          { label: 'ASCII', value: 'ascii' },
          { label: 'HEX', value: 'hex' },
        ]"
        size="small"
        style="width: 100px"
      />
      <NInput
        v-model:value="sendData"
        placeholder="输入要发送的数据"
        size="small"
        @keyup.enter="handleSend"
      />
      <NButton
        type="primary"
        size="small"
        :disabled="!sendData"
        @click="handleSend"
      >
        发送
      </NButton>
    </NSpace>
  </div>
</template>

<style scoped>
.send-panel {
  padding: 8px 12px;
  background: #252526;
}
</style>
