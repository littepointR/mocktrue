package serial

import (
	"testing"

	"github.com/suyue/mocktrue/internal/core/eventbus"
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
