package buffer

import (
	"sort"
	"sync"
)

// IndexEntry represents a point in the data stream for fast seeking.
type IndexEntry struct {
	Offset  int64
	Ts      int64 // nanosecond timestamp
	FrameNo int64 // frame sequence number (0 if not applicable)
}

// Index provides sparse indexing over the data stream for fast timestamp
// and frame-number based seeking. All methods are safe for concurrent use.
type Index struct {
	mu      sync.RWMutex
	entries []IndexEntry
}

// NewIndex constructs an empty index.
func NewIndex() *Index {
	return &Index{
		entries: make([]IndexEntry, 0, 1024),
	}
}

// Insert adds an entry. Entries must be inserted in offset order.
func (idx *Index) Insert(entry IndexEntry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries = append(idx.entries, entry)
}

// FindByTs returns the entry with the largest Ts <= ts, or (zero, false)
// if no such entry exists.
func (idx *Index) FindByTs(ts int64) (IndexEntry, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.entries) == 0 {
		return IndexEntry{}, false
	}

	// Binary search for the last entry with Ts <= ts
	i := sort.Search(len(idx.entries), func(i int) bool {
		return idx.entries[i].Ts > ts
	})

	if i == 0 {
		return IndexEntry{}, false
	}
	return idx.entries[i-1], true
}

// FindByFrameNo returns the entry with the given frame number, or (zero,
// false) if not found.
func (idx *Index) FindByFrameNo(frameNo int64) (IndexEntry, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	for _, entry := range idx.entries {
		if entry.FrameNo == frameNo {
			return entry, true
		}
	}
	return IndexEntry{}, false
}

// Reset clears the index.
func (idx *Index) Reset() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries = idx.entries[:0]
}
