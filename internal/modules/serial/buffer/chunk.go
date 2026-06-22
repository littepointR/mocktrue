package buffer

// Chunk represents an immutable block of received bytes.
// Once created, its Data slice must not be modified.
type Chunk struct {
	Seq        int64  // monotonic sequence number
	BaseOffset int64  // byte offset from the start of the stream
	Timestamp  string // capture time for this received byte block, RFC3339Nano/local when available
	Data       []byte // received bytes (copy, safe to hold)
}

// SnapshotChunk is one retained received block that overlaps a Snapshot range.
type SnapshotChunk struct {
	Offset    int64  // byte offset from the start of the stream
	Timestamp string // capture time for this block, empty when unknown
	Data      []byte // bytes in this block slice (copy, safe to hold)
}

// Snapshot is an immutable view over a requested byte range.
type Snapshot struct {
	Offset int64           // requested start offset
	Data   []byte          // bytes in [Offset, Offset+len(Data))
	Chunks []SnapshotChunk // chunk metadata overlapping the returned Data range
	Total  int64           // total bytes seen by the stream
	EOF    bool            // true if offset >= Total
}
