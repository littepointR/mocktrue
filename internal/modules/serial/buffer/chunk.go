package buffer

// Chunk represents an immutable block of received bytes.
// Once created, its Data slice must not be modified.
type Chunk struct {
	Seq        int64  // monotonic sequence number
	BaseOffset int64  // byte offset from the start of the stream
	Data       []byte // received bytes (copy, safe to hold)
}

// Snapshot is an immutable view over a requested byte range.
type Snapshot struct {
	Offset int64  // requested start offset
	Data   []byte // bytes in [Offset, Offset+len(Data))
	Total  int64  // total bytes in the buffer
	EOF    bool   // true if offset >= Total
}
