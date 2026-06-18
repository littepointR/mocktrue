package protocol

// FrameSpec defines a frame format for the visual parser.
type FrameSpec struct {
	Name        string         `json:"name"`
	Header      []byte         `json:"header,omitempty"`
	Footer      []byte         `json:"footer,omitempty"`
	LengthField *LengthField   `json:"lengthField,omitempty"`
	Checksum    *ChecksumSpec  `json:"checksum,omitempty"`
	Fields      []FieldSpec    `json:"fields"`
	MaxFrameLen int            `json:"maxFrameLen"`
	MinFrameLen int            `json:"minFrameLen"`
}

// LengthField describes how the frame length is encoded.
type LengthField struct {
	Offset           int    `json:"offset"`
	Width            int    `json:"width"` // 1/2/4
	Endianness       string `json:"endianness"` // "le"|"be"
	IncludesHeader   bool   `json:"includesHeader"`
	IncludesChecksum bool   `json:"includesChecksum"`
}

// ChecksumSpec describes the checksum algorithm.
type ChecksumSpec struct {
	Type   string `json:"type"` // "none"|"sum8"|"crc8"|"crc16"|"crc32"
	Offset int    `json:"offset"`
	Width  int    `json:"width"` // 1/2/4
	Poly   uint32 `json:"poly,omitempty"`
	Init   uint32 `json:"init,omitempty"`
	XorOut uint32 `json:"xorOut,omitempty"`
}

// FieldSpec describes a field within a frame.
type FieldSpec struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
	Width  int    `json:"width"`
	Type   string `json:"type"` // "u8"|"u16"|"u32"|"bytes"|"ascii"
}
