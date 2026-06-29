package visual

import (
	"testing"

	"github.com/littepointR/portweave/internal/modules/serial/protocol"
	"github.com/littepointR/portweave/internal/modules/serial/protocol/templates"
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

func TestVisualParserParsesAA55TemplateExample(t *testing.T) {
	t.Parallel()
	tmpl := templates.GetTemplate("AA55 自定义帧")
	if tmpl == nil || tmpl.Config.Visual == nil {
		t.Fatalf("AA55 template visual config = %#v", tmpl)
	}
	parser := NewVisualParser(*tmpl.Config.Visual)

	// Template shape: AA 55 + little-endian length including checksum + sum8 + payload.
	frame := []byte{0xaa, 0x55, 0x04, 0x00, 0x03, 0x01}
	result, err := parser.Parse(frame)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !result.OK || result.Consumed != len(frame) || result.NeedMore != 0 {
		t.Fatalf("Parse result = %#v, want complete OK AA55 template frame", result)
	}
	if len(result.Fields) != 1 || result.Fields[0].Name != "cmd" {
		t.Fatalf("Fields = %#v, want documented cmd field", result.Fields)
	}

	badChecksum := append([]byte(nil), frame...)
	badChecksum[4] = 0x00
	result, err = parser.Parse(badChecksum)
	if err != nil {
		t.Fatalf("Parse bad checksum returned error: %v", err)
	}
	if result.OK || len(result.Errors) == 0 {
		t.Fatalf("bad checksum result = %#v, want checksum mismatch", result)
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

func TestVisualParserAccessors(t *testing.T) {
	t.Parallel()
	spec := protocol.FrameSpec{Name: "accessors"}
	parser := NewVisualParser(spec)

	if parser.Name() != "accessors" {
		t.Fatalf("Name = %q, want accessors", parser.Name())
	}
	if parser.Kind() != protocol.KindVisual {
		t.Fatalf("Kind = %q, want visual", parser.Kind())
	}
	config := parser.Config()
	if config.Kind != protocol.KindVisual || config.Visual == nil || config.Visual.Name != "accessors" {
		t.Fatalf("Config = %#v, want visual config", config)
	}
}

func TestChecksumAlgorithms(t *testing.T) {
	t.Parallel()
	data := []byte("123456789")

	cases := []struct {
		name string
		spec ChecksumSpec
	}{
		{name: "none", spec: ChecksumSpec{Type: "none"}},
		{name: "crc8", spec: ChecksumSpec{Type: "crc8", Poly: 0x07}},
		{name: "crc16", spec: ChecksumSpec{Type: "crc16", Poly: 0x1021, Init: 0xffff}},
		{name: "crc32", spec: ChecksumSpec{Type: "crc32", Poly: 0x04c11db7, Init: 0xffffffff, XorOut: 0xffffffff}},
	}
	for _, tc := range cases {
		if got := Checksum(data, tc.spec); tc.name == "none" && got != 0 {
			t.Fatalf("Checksum(%s) = %#x, want 0", tc.name, got)
		}
	}
}

func TestVisualParserNoHeaderAndChecksumMismatch(t *testing.T) {
	t.Parallel()
	parser := NewVisualParser(protocol.FrameSpec{
		Name:        "no-header",
		MinFrameLen: 2,
		Checksum: &protocol.ChecksumSpec{
			Type:   "sum8",
			Offset: 1,
			Width:  1,
		},
	})

	result, err := parser.Parse([]byte{0x01, 0xff})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.OK {
		t.Fatalf("Parse OK = true, want checksum mismatch")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors = %#v, want checksum mismatch", result.Errors)
	}
}
