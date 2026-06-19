export function decodeSerialText(bytes: Uint8Array, encoding: string): string {
  const normalized = normalizeTextEncoding(encoding)
  if (normalized === 'ascii') {
    return Array.from(bytes, byte => byte <= 0x7f ? String.fromCharCode(byte) : '\ufffd').join('')
  }

  try {
    return new TextDecoder(textDecoderLabel(normalized)).decode(bytes)
  } catch {
    return new TextDecoder('utf-8').decode(bytes)
  }
}

export function normalizeTextEncoding(encoding: string): string {
  return (encoding || 'utf-8').trim().toLowerCase().replace(/_/g, '-')
}

function textDecoderLabel(encoding: string): string {
  switch (encoding) {
    case 'gb2312':
      return 'gbk'
    case 'shift-jis':
      return 'shift_jis'
    default:
      return encoding
  }
}
