package buffer

import (
	"testing"
)

func TestRingBufferAppendAndTotal(t *testing.T) {
	t.Parallel()
	r := NewRing(1024 * 1024) // 1MB cap
	if r.Total() != 0 {
		t.Fatalf("empty Total = %d, want 0", r.Total())
	}

	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: []byte("hello")})
	if r.Total() != 5 {
		t.Fatalf("Total after 'hello' = %d, want 5", r.Total())
	}

	r.Append(Chunk{Seq: 1, BaseOffset: 5, Data: []byte("world")})
	if r.Total() != 10 {
		t.Fatalf("Total after 'world' = %d, want 10", r.Total())
	}
}

func TestRingBufferQueryExactRange(t *testing.T) {
	t.Parallel()
	r := NewRing(1024 * 1024)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: []byte("0123456789")})
	r.Append(Chunk{Seq: 1, BaseOffset: 10, Data: []byte("abcdefghij")})

	snap, err := r.Query(5, 5) // should get "56789"
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if string(snap.Data) != "56789" {
		t.Fatalf("Query(5,5) = %q, want %q", snap.Data, "56789")
	}
	if snap.EOF {
		t.Fatalf("should not be EOF")
	}
}

func TestRingBufferQueryCrossesChunkBoundary(t *testing.T) {
	t.Parallel()
	r := NewRing(1024 * 1024)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: []byte("01234")})
	r.Append(Chunk{Seq: 1, BaseOffset: 5, Data: []byte("56789")})

	snap, err := r.Query(3, 5) // "34" + "567"
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if string(snap.Data) != "34567" {
		t.Fatalf("Query(3,5) = %q, want %q", snap.Data, "34567")
	}
}

func TestRingBufferQueryReturnsChunkMetadata(t *testing.T) {
	t.Parallel()
	r := NewRing(1024 * 1024)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Timestamp: "2026-06-23T02:01:02.003", Data: []byte("hello")})
	r.Append(Chunk{Seq: 1, BaseOffset: 5, Timestamp: "2026-06-23T02:01:03.004", Data: []byte("world")})

	snap, err := r.Query(3, 4)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if string(snap.Data) != "lowo" {
		t.Fatalf("Query(3,4) = %q, want %q", snap.Data, "lowo")
	}
	if len(snap.Chunks) != 2 {
		t.Fatalf("len(Chunks) = %d, want 2", len(snap.Chunks))
	}
	if snap.Chunks[0].Offset != 3 || snap.Chunks[0].Timestamp != "2026-06-23T02:01:02.003" || string(snap.Chunks[0].Data) != "lo" {
		t.Fatalf("Chunks[0] = %+v, want offset 3 timestamp 2026-06-23T02:01:02.003 data lo", snap.Chunks[0])
	}
	if snap.Chunks[1].Offset != 5 || snap.Chunks[1].Timestamp != "2026-06-23T02:01:03.004" || string(snap.Chunks[1].Data) != "wo" {
		t.Fatalf("Chunks[1] = %+v, want offset 5 timestamp 2026-06-23T02:01:03.004 data wo", snap.Chunks[1])
	}
}

func TestRingBufferQueryEOF(t *testing.T) {
	t.Parallel()
	r := NewRing(1024 * 1024)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: []byte("abc")})

	snap, err := r.Query(3, 10) // offset at end
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !snap.EOF {
		t.Fatalf("should be EOF at offset 3")
	}
	if len(snap.Data) != 0 {
		t.Fatalf("EOF data should be empty, got %d bytes", len(snap.Data))
	}
}

func TestRingBufferQueryClampsToTotal(t *testing.T) {
	t.Parallel()
	r := NewRing(1024 * 1024)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: []byte("abcde")})

	snap, err := r.Query(2, 100) // ask for 100 bytes starting at 2, but only 5 total
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if string(snap.Data) != "cde" {
		t.Fatalf("Query(2,100) = %q, want %q", snap.Data, "cde")
	}
	if snap.EOF {
		t.Fatalf("should not be EOF (still have bytes)")
	}
}

func TestRingBufferReset(t *testing.T) {
	t.Parallel()
	r := NewRing(1024 * 1024)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: []byte("abc")})
	r.Reset()
	if r.Total() != 0 {
		t.Fatalf("Total after Reset = %d, want 0", r.Total())
	}
}

func TestRingBufferQueryBeforeRetainedWindow(t *testing.T) {
	t.Parallel()
	r := NewRing(1024)
	data := make([]byte, 512)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: data})
	r.Append(Chunk{Seq: 1, BaseOffset: 512, Data: data})
	r.Append(Chunk{Seq: 2, BaseOffset: 1024, Data: data})

	snap, err := r.Query(0, 100)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(snap.Data) != 0 {
		t.Fatalf("Query before retained window returned %d bytes, want 0", len(snap.Data))
	}
	if snap.Total != 1536 {
		t.Fatalf("Query total = %d, want lifetime total 1536", snap.Total)
	}
}

func TestRingBufferQueryOversizedChunkIsCapped(t *testing.T) {
	t.Parallel()
	r := NewRing(4)
	r.Append(Chunk{Seq: 0, BaseOffset: 0, Data: []byte("abcdef")})

	if r.Total() != 6 {
		t.Fatalf("Total = %d, want lifetime total 6", r.Total())
	}
	snap, err := r.Query(2, 4)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if string(snap.Data) != "cdef" {
		t.Fatalf("Query retained oversized data = %q, want cdef", string(snap.Data))
	}
}
