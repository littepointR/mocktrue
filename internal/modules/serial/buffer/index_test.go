package buffer

import (
	"testing"
)

func TestRingBufferSpillToDisk(t *testing.T) {
	t.Parallel()
	// Use a small capacity to force eviction of old chunks
	r := NewRing(1024) // 1KB cap

	// Append more than capacity
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i % 256)
	}

	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: data})
	r.Append(Chunk{Seq: 1, BaseOffset: 512, Data: data})
	r.Append(Chunk{Seq: 2, BaseOffset: 1024, Data: data}) // should evict oldest

	// Old chunk (512 bytes) is evicted from storage, but Total tracks the
	// lifetime byte count so virtual scrollers can keep growing.
	if r.Total() != 1536 {
		t.Fatalf("Total = %d, want 1536 lifetime bytes", r.Total())
	}

	// Query should still work on remaining chunks
	snap, err := r.Query(512, 100)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(snap.Data) != 100 {
		t.Fatalf("Query returned %d bytes, want 100", len(snap.Data))
	}
}

func TestIndexInsertAndFind(t *testing.T) {
	t.Parallel()
	idx := NewIndex()

	idx.Insert(IndexEntry{Offset: 0, Ts: 1000, FrameNo: 0})
	idx.Insert(IndexEntry{Offset: 100, Ts: 2000, FrameNo: 1})
	idx.Insert(IndexEntry{Offset: 200, Ts: 3000, FrameNo: 2})

	// Find by timestamp
	entry, found := idx.FindByTs(1500)
	if !found {
		t.Fatalf("FindByTs(1500) not found")
	}
	if entry.Offset != 0 {
		t.Fatalf("FindByTs(1500) = offset %d, want 0", entry.Offset)
	}

	entry, found = idx.FindByTs(2500)
	if !found {
		t.Fatalf("FindByTs(2500) not found")
	}
	if entry.Offset != 100 {
		t.Fatalf("FindByTs(2500) = offset %d, want 100", entry.Offset)
	}

	// Find by frame number
	entry, found = idx.FindByFrameNo(2)
	if !found {
		t.Fatalf("FindByFrameNo(2) not found")
	}
	if entry.Offset != 200 {
		t.Fatalf("FindByFrameNo(2) = offset %d, want 200", entry.Offset)
	}
}
