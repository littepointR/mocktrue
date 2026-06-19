import { Events } from '@wailsio/runtime'

export interface MCPStatus {
  Enabled: boolean
  Running: boolean
  Address: string
  Path: string
  Error: string
}

export function onMCPStatus(callback: (status: MCPStatus) => void): () => void {
  return Events.On('mcp:status', (event: { data: MCPStatus }) => {
    callback(event.data)
  })
}
