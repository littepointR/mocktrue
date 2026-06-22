package buffer

import (
	"sync"
)

// RingBuffer is a pure in-memory ring buffer for serial data. Stage 1 uses
// this; stage 2 will add spill-to-disk for >256MB.
//
// All methods are safe for concurrent use.
type RingBuffer struct {
	mu       sync.RWMutex
	chunks   []Chunk
	total    int64 // lifetime byte count
	retained int64 // bytes currently retained in chunks
	capacity int64 // max bytes before old chunks are evicted
}

// NewRing constructs a ring buffer with the given byte capacity.
func NewRing(capacity int64) *RingBuffer {
	return &RingBuffer{
		chunks:   make([]Chunk, 0, 64),
		capacity: capacity,
	}
}

// Append adds a chunk. Old chunks are evicted if the buffer exceeds capacity.
// The chunk's Data slice is assumed to be immutable after this call.
func (r *RingBuffer) Append(c Chunk) {
	r.mu.Lock()
	defer r.mu.Unlock()

	dataLen := int64(len(c.Data))
	end := c.BaseOffset + dataLen
	if end > r.total {
		r.total = end
	}
	if dataLen == 0 || r.capacity <= 0 {
		return
	}
	if dataLen > r.capacity {
		drop := int(dataLen - r.capacity)
		c.BaseOffset += int64(drop)
		c.Data = c.Data[drop:]
		dataLen = int64(len(c.Data))
	}

	r.chunks = append(r.chunks, c)
	r.retained += dataLen

	for r.retained > r.capacity && len(r.chunks) > 0 {
		excess := r.retained - r.capacity
		oldest := r.chunks[0]
		oldestLen := int64(len(oldest.Data))
		if oldestLen <= excess {
			r.retained -= oldestLen
			r.chunks = r.chunks[1:]
			continue
		}
		drop := int(excess)
		oldest.BaseOffset += excess
		oldest.Data = oldest.Data[drop:]
		r.chunks[0] = oldest
		r.retained -= excess
	}
}

// Query returns a snapshot of the bytes in [offset, offset+length). If
// offset >= Total, EOF is set. The returned Data is a copy (safe to hold).
func (r *RingBuffer) Query(offset int64, length int) (*Snapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset < 0 {
		offset = 0
	}
	if length <= 0 {
		return &Snapshot{Offset: offset, Total: r.total, EOF: offset >= r.total}, nil
	}
	if offset >= r.total {
		return &Snapshot{Offset: offset, Total: r.total, EOF: true}, nil
	}
	if len(r.chunks) == 0 || offset < r.chunks[0].BaseOffset {
		return &Snapshot{Offset: offset, Total: r.total, EOF: false}, nil
	}

	// Clamp length to available bytes
	if offset+int64(length) > r.total {
		length = int(r.total - offset)
	}

	// Find the starting chunk (binary search would be better for many chunks)
	startChunk := 0
	startOffset := offset
	found := false
	for i, c := range r.chunks {
		if c.BaseOffset+int64(len(c.Data)) > offset {
			startChunk = i
			startOffset = offset - c.BaseOffset
			found = true
			break
		}
	}
	if !found || startOffset < 0 {
		return &Snapshot{Offset: offset, Total: r.total, EOF: false}, nil
	}

	// Copy bytes across chunks
	result := make([]byte, 0, length)
	snapshotChunks := make([]SnapshotChunk, 0, len(r.chunks)-startChunk)
	remaining := length
	chunkIdx := startChunk
	chunkOff := startOffset

	for remaining > 0 && chunkIdx < len(r.chunks) {
		c := r.chunks[chunkIdx]
		available := int64(len(c.Data)) - chunkOff
		if available > int64(remaining) {
			available = int64(remaining)
		}
		chunkData := append([]byte(nil), c.Data[chunkOff:chunkOff+available]...)
		result = append(result, chunkData...)
		snapshotChunks = append(snapshotChunks, SnapshotChunk{
			Offset:    c.BaseOffset + chunkOff,
			Timestamp: c.Timestamp,
			Data:      chunkData,
		})
		remaining -= int(available)
		chunkIdx++
		chunkOff = 0
	}

	return &Snapshot{
		Offset: offset,
		Data:   result,
		Chunks: snapshotChunks,
		Total:  r.total,
		EOF:    false,
	}, nil
}

// Total returns the total number of bytes in the buffer.
func (r *RingBuffer) Total() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.total
}

// Reset clears the buffer.
func (r *RingBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chunks = r.chunks[:0]
	r.total = 0
	r.retained = 0
}
