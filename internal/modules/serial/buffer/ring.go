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
	total    int64
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

	// Evict old chunks if we exceed capacity
	for r.total+int64(len(c.Data)) > r.capacity && len(r.chunks) > 0 {
		oldest := r.chunks[0]
		r.total -= int64(len(oldest.Data))
		r.chunks = r.chunks[1:]
	}

	r.chunks = append(r.chunks, c)
	r.total += int64(len(c.Data))
}

// Query returns a snapshot of the bytes in [offset, offset+length). If
// offset >= Total, EOF is set. The returned Data is a copy (safe to hold).
func (r *RingBuffer) Query(offset int64, length int) (*Snapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset >= r.total {
		return &Snapshot{Offset: offset, Total: r.total, EOF: true}, nil
	}

	// Clamp length to available bytes
	if offset+int64(length) > r.total {
		length = int(r.total - offset)
	}

	// Find the starting chunk (binary search would be better for many chunks)
	startChunk := 0
	startOffset := offset
	for i, c := range r.chunks {
		if c.BaseOffset+int64(len(c.Data)) > offset {
			startChunk = i
			startOffset = offset - c.BaseOffset
			break
		}
	}

	// Copy bytes across chunks
	result := make([]byte, 0, length)
	remaining := length
	chunkIdx := startChunk
	chunkOff := startOffset

	for remaining > 0 && chunkIdx < len(r.chunks) {
		c := r.chunks[chunkIdx]
		available := int64(len(c.Data)) - chunkOff
		if available > int64(remaining) {
			available = int64(remaining)
		}
		result = append(result, c.Data[chunkOff:chunkOff+available]...)
		remaining -= int(available)
		chunkIdx++
		chunkOff = 0
	}

	return &Snapshot{
		Offset: offset,
		Data:   result,
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
}
