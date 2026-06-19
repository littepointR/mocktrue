export type BytePayload = number[] | Uint8Array | string | null | undefined

export function toByteArray(payload: BytePayload): number[] {
  if (!payload) return []
  if (typeof payload === 'string') {
    return Array.from(atob(payload), char => char.charCodeAt(0))
  }
  return Array.from(payload)
}
