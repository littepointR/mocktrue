package templates

import (
	"github.com/littepointR/mocktrue/internal/modules/serial/protocol"
)

// Template represents a predefined protocol template.
type Template struct {
	Name        string
	Description string
	Kind        protocol.Kind
	Config      protocol.ParserConfig
}

// Registry holds all available protocol templates.
var Registry = []Template{
	{
		Name:        "Modbus RTU",
		Description: "标准 Modbus RTU 帧格式，支持读/写功能码",
		Kind:        protocol.KindVisual,
		Config: protocol.ParserConfig{
			Kind: protocol.KindVisual,
			Visual: &protocol.FrameSpec{
				Name:        "Modbus RTU",
				Header:      []byte{}, // 无固定帧头
				LengthField: nil,      // 长度由功能码决定
				Checksum: &protocol.ChecksumSpec{
					Type:   "crc16",
					Offset: -2, // 校验在帧尾
					Width:  2,
					Poly:   0xA001,
					Init:   0xFFFF,
					XorOut: 0x0000,
				},
				Fields: []protocol.FieldSpec{
					{Name: "SlaveAddr", Offset: 0, Width: 1, Type: "u8"},
					{Name: "FuncCode", Offset: 1, Width: 1, Type: "u8"},
				},
				MaxFrameLen: 256,
				MinFrameLen: 4, // addr + func + crc16
			},
		},
	},
	{
		Name:        "AA55 自定义帧",
		Description: "常见嵌入式自定义帧格式: AA 55 + 长度 + 数据 + 校验",
		Kind:        protocol.KindVisual,
		Config: protocol.ParserConfig{
			Kind: protocol.KindVisual,
			Visual: &protocol.FrameSpec{
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
					{Name: "cmd", Offset: 4, Width: 1, Type: "u8"},
				},
				MaxFrameLen: 1024,
				MinFrameLen: 5,
			},
		},
	},
	{
		Name:        "NMEA",
		Description: "NMEA 0183 文本协议 ($...*CS)，用于 GPS/导航设备",
		Kind:        protocol.KindScript,
		Config: protocol.ParserConfig{
			Kind:   protocol.KindScript,
			Script: nmeaScript,
		},
	},
}

const nmeaScript = `
// NMEA 0183 parser
// Format: $<sentence>*<checksum>\r\n
var str = "";
for (var i = 0; i < len(); i++) {
    var b = byte(i);
    if (b == 0x24) { // '$'
        // Start of sentence
        var start = i;
        var end = -1;
        for (var j = i + 1; j < len(); j++) {
            if (byte(j) == 0x2A) { // '*'
                end = j;
                break;
            }
        }
        if (end > 0 && end + 2 < len()) {
            var sentence = "";
            for (var k = start + 1; k < end; k++) {
                sentence += String.fromCharCode(byte(k));
            }
            field("sentence", sentence);
            var cs = 0;
            for (var k = start + 1; k < end; k++) {
                cs ^= byte(k);
            }
            var recvCs = parseInt(String.fromCharCode(byte(end + 1)) + String.fromCharCode(byte(end + 2)), 16);
            if (cs === recvCs) {
                field("checksum", "0x" + cs.toString(16), "OK");
            } else {
                error("checksum mismatch: expected 0x" + cs.toString(16) + ", got 0x" + recvCs.toString(16));
            }
        }
        break;
    }
}
`

// GetTemplate returns a template by name, or nil if not found.
func GetTemplate(name string) *Template {
	for _, t := range Registry {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// ListTemplates returns the names and descriptions of all available templates.
func ListTemplates() []struct {
	Name        string
	Description string
	Kind        protocol.Kind
} {
	result := make([]struct {
		Name        string
		Description string
		Kind        protocol.Kind
	}, len(Registry))
	for i, t := range Registry {
		result[i] = struct {
			Name        string
			Description string
			Kind        protocol.Kind
		}{t.Name, t.Description, t.Kind}
	}
	return result
}
