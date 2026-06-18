package visual

import (
	"fmt"

	"github.com/suyue/mocktrue/internal/modules/serial/protocol"
)

// VisualParser parses frames based on a FrameSpec configuration.
type VisualParser struct {
	spec protocol.FrameSpec
}

// NewVisualParser creates a parser from a FrameSpec.
func NewVisualParser(spec protocol.FrameSpec) *VisualParser {
	return &VisualParser{spec: spec}
}

// Name returns the parser name.
func (p *VisualParser) Name() string { return p.spec.Name }

// Kind returns KindVisual.
func (p *VisualParser) Kind() protocol.Kind { return protocol.KindVisual }

// Config returns the serializable configuration.
func (p *VisualParser) Config() protocol.ParserConfig {
	return protocol.ParserConfig{
		Kind:   protocol.KindVisual,
		Visual: &p.spec,
	}
}

// Parse attempts to parse a frame from the input data.
func (p *VisualParser) Parse(data []byte) (protocol.ParseResult, error) {
	if len(data) == 0 {
		return protocol.ParseResult{NeedMore: 1}, nil
	}

	// Find header
	headerStart := -1
	if len(p.spec.Header) == 0 {
		headerStart = 0
	} else {
		for i := 0; i <= len(data)-len(p.spec.Header); i++ {
			match := true
			for j, b := range p.spec.Header {
				if data[i+j] != b {
					match = false
					break
				}
			}
			if match {
				headerStart = i
				break
			}
		}
	}

	if headerStart < 0 {
		return protocol.ParseResult{NeedMore: len(p.spec.Header)}, nil
	}

	// Determine frame length
	frameLen := 0
	if p.spec.LengthField != nil && p.spec.LengthField.Width > 0 {
		lf := p.spec.LengthField
		if headerStart+lf.Offset+lf.Width > len(data) {
			return protocol.ParseResult{NeedMore: headerStart + lf.Offset + lf.Width - len(data)}, nil
		}
		lengthBytes := data[headerStart+lf.Offset : headerStart+lf.Offset+lf.Width]
		lengthValue := decodeUint(lengthBytes, lf.Endianness == "le")
		frameLen = int(lengthValue)
		if p.spec.LengthField.IncludesHeader {
			// length already includes header
		} else {
			frameLen += len(p.spec.Header)
		}
		if p.spec.LengthField.IncludesChecksum {
			// length already includes checksum
		} else if p.spec.Checksum != nil && p.spec.Checksum.Width > 0 {
			frameLen += p.spec.Checksum.Width
		}
	} else {
		// No length field: use MinFrameLen or header+len+footer
		frameLen = p.spec.MinFrameLen
		if frameLen == 0 {
			frameLen = len(p.spec.Header) + 1 // minimum: header + 1 byte
		}
	}

	if frameLen > p.spec.MaxFrameLen && p.spec.MaxFrameLen > 0 {
		frameLen = p.spec.MaxFrameLen
	}

	// Check if we have enough data
	if headerStart+frameLen > len(data) {
		return protocol.ParseResult{NeedMore: headerStart + frameLen - len(data)}, nil
	}

	frameData := data[headerStart : headerStart+frameLen]

	// Verify checksum if present
	if p.spec.Checksum != nil && p.spec.Checksum.Width > 0 {
		cs := p.spec.Checksum
		if cs.Offset+cs.Width <= len(frameData) {
			expected := decodeUint(frameData[cs.Offset:cs.Offset+cs.Width], true)
			checksumData := frameData[:cs.Offset]
			actual := Checksum(checksumData, ChecksumSpec{
				Type:   cs.Type,
				Offset: cs.Offset,
				Width:  cs.Width,
				Poly:   cs.Poly,
				Init:   cs.Init,
				XorOut: cs.XorOut,
			})
			if uint32(expected) != actual {
				return protocol.ParseResult{
					OK:       false,
					Errors:   []string{fmt.Sprintf("checksum mismatch: expected 0x%02x, got 0x%02x", expected, actual)},
					Consumed: headerStart + frameLen,
				}, nil
			}
		}
	}

	// Extract fields
	fields := make([]protocol.Field, 0, len(p.spec.Fields))
	for _, fs := range p.spec.Fields {
		if fs.Offset+fs.Width > len(frameData) {
			continue
		}
		fieldData := frameData[fs.Offset : fs.Offset+fs.Width]
		value := decodeUint(fieldData, true) // default LE
		fields = append(fields, protocol.Field{
			Name:    fs.Name,
			Offset:  fs.Offset,
			Width:   fs.Width,
			Type:    fs.Type,
			Value:   value,
			Display: fmt.Sprintf("0x%0*x", fs.Width*2, value),
		})
	}

	return protocol.ParseResult{
		OK:       true,
		Fields:   fields,
		Consumed: headerStart + frameLen,
	}, nil
}

func decodeUint(data []byte, littleEndian bool) uint64 {
	var result uint64
	if littleEndian {
		for i := len(data) - 1; i >= 0; i-- {
			result = result<<8 | uint64(data[i])
		}
	} else {
		for _, b := range data {
			result = result<<8 | uint64(b)
		}
	}
	return result
}
