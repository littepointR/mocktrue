package protocol

import (
	"testing"
)

func TestFrameSpecStruct(t *testing.T) {
	t.Parallel()
	spec := FrameSpec{
		Name:   "TestFrame",
		Header: []byte{0xAA, 0x55},
		LengthField: &LengthField{
			Offset:     2,
			Width:      2,
			Endianness: "le",
		},
		Checksum: &ChecksumSpec{
			Type:   "sum8",
			Offset: 4,
			Width:  1,
		},
		Fields: []FieldSpec{
			{Name: "cmd", Offset: 0, Width: 1, Type: "u8"},
		},
		MaxFrameLen: 256,
	}

	if spec.Name != "TestFrame" {
		t.Fatalf("Name = %q, want TestFrame", spec.Name)
	}
	if len(spec.Header) != 2 {
		t.Fatalf("Header len = %d, want 2", len(spec.Header))
	}
	if spec.LengthField == nil || spec.LengthField.Width != 2 {
		t.Fatalf("LengthField missing or wrong width")
	}
}

func TestParseResultStruct(t *testing.T) {
	t.Parallel()
	result := ParseResult{
		OK: true,
		Fields: []Field{
			{Name: "cmd", Offset: 0, Width: 1, Value: uint8(0x01), Display: "0x01"},
		},
		Consumed: 10,
		NeedMore: 0,
	}

	if !result.OK {
		t.Fatalf("OK should be true")
	}
	if len(result.Fields) != 1 {
		t.Fatalf("Fields len = %d, want 1", len(result.Fields))
	}
	if result.Fields[0].Name != "cmd" {
		t.Fatalf("Field name = %q, want cmd", result.Fields[0].Name)
	}
}
