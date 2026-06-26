package serial

import (
	"testing"

	"github.com/littepointR/portweave/internal/core/eventbus"
)

func TestEncodeSerialTextSupportsConfiguredEncodings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		encoding string
		text     string
		want     []byte
	}{
		{name: "utf8", encoding: "utf-8", text: "串口", want: []byte{0xe4, 0xb8, 0xb2, 0xe5, 0x8f, 0xa3}},
		{name: "utf16le", encoding: "utf-16le", text: "串口", want: []byte{0x32, 0x4e, 0xe3, 0x53}},
		{name: "utf16be", encoding: "utf-16be", text: "串口", want: []byte{0x4e, 0x32, 0x53, 0xe3}},
		{name: "gbk", encoding: "gbk", text: "串口", want: []byte{0xb4, 0xae, 0xbf, 0xda}},
		{name: "gb2312", encoding: "gb2312", text: "串口", want: []byte{0xb4, 0xae, 0xbf, 0xda}},
		{name: "big5", encoding: "big5", text: "串口", want: []byte{0xa6, 0xea, 0xa4, 0x66}},
		{name: "shift_jis", encoding: "shift_jis", text: "カナ", want: []byte{0x83, 0x4a, 0x83, 0x69}},
		{name: "windows1251", encoding: "windows-1251", text: "Б", want: []byte{0xc1}},
		{name: "windows1252", encoding: "windows-1252", text: "é", want: []byte{0xe9}},
		{name: "ascii", encoding: "ascii", text: "ABC", want: []byte{0x41, 0x42, 0x43}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := encodeSerialText(tt.text, tt.encoding)
			if err != nil {
				t.Fatalf("encodeSerialText returned error: %v", err)
			}
			if string(got) != string(tt.want) {
				t.Fatalf("encodeSerialText(%q, %q) = % x, want % x", tt.text, tt.encoding, got, tt.want)
			}
		})
	}
}

func TestEncodeSerialTextRejectsUnsupportedEncoding(t *testing.T) {
	t.Parallel()

	if _, err := encodeSerialText("abc", "unsupported"); err == nil {
		t.Fatalf("encodeSerialText must reject unsupported encodings")
	}
}

func TestDecodeSerialTextSupportsConfiguredEncodings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		encoding string
		data     []byte
		want     string
	}{
		{name: "utf8", encoding: "utf-8", data: []byte{0xe4, 0xb8, 0xb2, 0xe5, 0x8f, 0xa3}, want: "串口"},
		{name: "utf16le", encoding: "utf-16le", data: []byte{0x32, 0x4e, 0xe3, 0x53}, want: "串口"},
		{name: "utf16be", encoding: "utf-16be", data: []byte{0x4e, 0x32, 0x53, 0xe3}, want: "串口"},
		{name: "gbk", encoding: "gbk", data: []byte{0xb4, 0xae, 0xbf, 0xda}, want: "串口"},
		{name: "gb2312", encoding: "gb2312", data: []byte{0xb4, 0xae, 0xbf, 0xda}, want: "串口"},
		{name: "gb18030", encoding: "gb18030", data: []byte{0xb4, 0xae, 0xbf, 0xda}, want: "串口"},
		{name: "big5", encoding: "big5", data: []byte{0xa6, 0xea, 0xa4, 0x66}, want: "串口"},
		{name: "shiftjis", encoding: "shift-jis", data: []byte{0x83, 0x4a, 0x83, 0x69}, want: "カナ"},
		{name: "windows1251", encoding: "windows1251", data: []byte{0xc1}, want: "Б"},
		{name: "windows1252", encoding: "cp1252", data: []byte{0xe9}, want: "é"},
		{name: "ascii replacement", encoding: "ascii", data: []byte{'A', 0xff, 'B'}, want: "A�B"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeSerialText(tt.data, tt.encoding)
			if err != nil {
				t.Fatalf("decodeSerialText returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("decodeSerialText(% x, %q) = %q, want %q", tt.data, tt.encoding, got, tt.want)
			}
		})
	}
}

func TestSerialTextEncodingRejectsASCIIAndUnsupportedDecodeErrors(t *testing.T) {
	t.Parallel()

	if _, err := encodeSerialText("é", "ascii"); err == nil {
		t.Fatalf("encodeSerialText must reject non-ASCII runes in ascii mode")
	}
	if _, err := decodeSerialText([]byte("abc"), "unsupported"); err == nil {
		t.Fatalf("decodeSerialText must reject unsupported encodings")
	}
}

func TestServiceTextConversionUsesSelectedEncoding(t *testing.T) {
	t.Parallel()

	svc := NewService(eventbus.New())
	hexText, err := svc.EncodeTextToHex(EncodeTextRequest{Content: "串口", Encoding: "gbk"})
	if err != nil {
		t.Fatalf("EncodeTextToHex returned error: %v", err)
	}
	if hexText != "b4 ae bf da" {
		t.Fatalf("EncodeTextToHex = %q, want b4 ae bf da", hexText)
	}

	text, err := svc.DecodeHexToText(DecodeHexRequest{Content: hexText, Encoding: "gbk"})
	if err != nil {
		t.Fatalf("DecodeHexToText returned error: %v", err)
	}
	if text != "串口" {
		t.Fatalf("DecodeHexToText = %q, want 串口", text)
	}
}

func TestServiceTextConversionRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	svc := NewService(eventbus.New())
	if _, err := svc.EncodeTextToHex(EncodeTextRequest{Content: "é", Encoding: "ascii"}); err == nil {
		t.Fatalf("EncodeTextToHex must reject non-ASCII text in ascii mode")
	}
	if _, err := svc.DecodeHexToText(DecodeHexRequest{Content: "zz", Encoding: "utf-8"}); err == nil {
		t.Fatalf("DecodeHexToText must reject malformed hex")
	}
	if _, err := svc.DecodeHexToText(DecodeHexRequest{Content: "41", Encoding: "unsupported"}); err == nil {
		t.Fatalf("DecodeHexToText must reject unsupported encodings")
	}
}
