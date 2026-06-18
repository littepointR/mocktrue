import { ref } from 'vue'

interface CacheEntry {
  data: Uint8Array
  lastAccess: number
}

const MAX_ENTRIES = 256 // 256 * 64KB = 16MB cache
const CHUNK_SIZE = 64 * 1024 // 64KB per chunk

/**
 * useBlobChunk provides an LRU cache for QueryPage results, aligned to
 * 64KB boundaries. Returns a function to fetch a byte range, using the
 * cache to avoid redundant requests.
 */
export function useBlobChunk(queryPage: (offset: number, length: number) => Promise<{ Data: string | Uint8Array | null }>) {
  const cache = new Map<string, CacheEntry>()

  function alignKey(offset: number): string {
    const aligned = Math.floor(offset / CHUNK_SIZE) * CHUNK_SIZE
    return `${aligned}`
  }

  async function fetch(offset: number, length: number): Promise<Uint8Array> {
    const result = new Uint8Array(length)
    let remaining = length
    let resultOffset = 0
    let fetchOffset = Math.floor(offset / CHUNK_SIZE) * CHUNK_SIZE

    while (remaining > 0) {
      const key = `${fetchOffset}`
      let entry = cache.get(key)

      if (!entry) {
        // Fetch from backend
        const snap = await queryPage(fetchOffset, CHUNK_SIZE)
        const rawData = snap.Data
        let data: Uint8Array
        if (rawData === null || rawData === undefined) {
          data = new Uint8Array(0)
        } else if (typeof rawData === 'string') {
          // Decode base64 string to Uint8Array
          data = new Uint8Array(Array.from(atob(rawData), c => c.charCodeAt(0)))
        } else {
          data = rawData
        }
        entry = { data, lastAccess: Date.now() }
        cache.set(key, entry)
      }

      entry.lastAccess = Date.now()

      // Copy from cache to result
      const chunkOffset = offset - fetchOffset
      const toCopy = Math.min(remaining, CHUNK_SIZE - chunkOffset)
      result.set(entry.data.slice(chunkOffset, chunkOffset + toCopy), resultOffset)

      resultOffset += toCopy
      remaining -= toCopy
      fetchOffset += CHUNK_SIZE
      offset = fetchOffset
    }

    // Evict oldest entries if cache is too large
    if (cache.size > MAX_ENTRIES) {
      const entries = Array.from(cache.entries()).sort((a, b) => a[1].lastAccess - b[1].lastAccess)
      const toEvict = entries.slice(0, entries.length - MAX_ENTRIES)
      for (const [key] of toEvict) {
        cache.delete(key)
      }
    }

    return result
  }

  function clear() {
    cache.clear()
  }

  return { fetch, clear }
}
