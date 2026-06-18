package visual

import (
	"testing"

	"github.com/suyue/mocktrue/internal/modules/serial/protocol"
)

func TestVisualParserAA55Frame(t *testing.T) {
	t.Parallel()
	spec := protocol.FrameSpec{
		Name:   "AA55",
		Header: []byte{0xAA, 0x55},
		LengthField: &protocol.LengthField{
			Offset:           2,
			Width:            2,
			Endianness:       "le",
			IncludesChecksum: true,
		},
		Checksum: &protocol.ChecksumSpec{
			Type:   "sum8",
			Offset: 4,
			Width:  1,
		},
		Fields: []protocol.FieldSpec{
			{Name: "cmd", Offset: 0, Width: 1, Type: "u8"},
			{Name: "data", Offset: 5, Width: 1, Type: "u8"},
		},
		MaxFrameLen: 256,
	}

	parser := NewVisualParser(spec)

	// Frame: AA 55 04 00 XX 01
	// Header(2) + Length(2) + Checksum(1) + Data(1) = 6 bytes
	// Length = 4 (LE) = 4 bytes after header
	frame := []byte{0xAA, 0x55, 0x04, 0x00, 0x00, 0x01}
	// Calculate checksum: sum of bytes before checksum (mod 256)
	frame[4] = byte((0xAA + 0x55 + 0x04 + 0x00) & 0xFF) // = 0x03

	result, err := parser.Parse(frame)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !result.OK {
		t.Fatalf("Parse not OK: %v", result.Errors)
	}
	if result.Consumed != 6 {
		t.Fatalf("Consumed = %d, want 6", result.Consumed)
	}
	if len(result.Fields) != 2 {
		t.Fatalf("Fields len = %d, want 2", len(result.Fields))
	}
}

func TestVisualParserNeedMore(t *testing.T) {
	t.Parallel()
	spec := protocol.FrameSpec{
		Name:   "Test",
		Header: []byte{0xAA},
		LengthField: &protocol.LengthField{
			Offset: 1,
			Width:  1,
		},
		MaxFrameLen: 256,
	}

	parser := NewVisualParser(spec)

	// Only header, not enough for length field
	result, err := parser.Parse([]byte{0xAA})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.NeedMore == 0 {
		t.Fatalf("should need more bytes")
	}
}

func TestChecksumSum8(t *testing.T) {
	t.Parallel()
	data := []byte{0x01, 0x02, 0x03}
	spec := ChecksumSpec{Type: "sum8"}
	result := Checksum(data, spec)
	if result != 6 {
		t.Fatalf("sum8 = %d, want 6", result)
	}
}
