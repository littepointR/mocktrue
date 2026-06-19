export type BytePayload = number[] | Uint8Array | string | null | undefined

export function toByteArray(payload: BytePayload): number[] {
  if (!payload) return []
  if (typeof payload === 'string') {
    return Array.from(atob(payload), char => char.charCodeAt(0))
  }
  return Array.from(payload)
}

export function formatHexInput(value: string): string {
  const normalized = value
    .replace(/0x/gi, '')
    .replace(/[^0-9a-fA-F]/g, '')
    .toLowerCase()
  const parts: string[] = []

  for (let index = 0; index < normalized.length; index += 2) {
    parts.push(normalized.slice(index, index + 2))
  }

  return parts.join(' ')
}

export function asciiToHexText(value: string): string {
  return formatBytesAsHex(Array.from(new TextEncoder().encode(value)))
}

export function hexTextToAscii(value: string): string {
  const bytes = hexTextToBytes(value)
  return new TextDecoder().decode(new Uint8Array(bytes))
}

function hexTextToBytes(value: string): number[] {
  const normalized = formatHexInput(value).replace(/\s+/g, '')
  if (!normalized) return []
  if (normalized.length % 2 !== 0) {
    throw new Error('hex input must contain complete bytes')
  }

  const bytes: number[] = []
  for (let index = 0; index < normalized.length; index += 2) {
    bytes.push(Number.parseInt(normalized.slice(index, index + 2), 16))
  }
  return bytes
}

function formatBytesAsHex(bytes: number[]): string {
  return bytes.map(byte => byte.toString(16).padStart(2, '0')).join(' ')
}
