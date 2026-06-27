package monitor

import "testing"

func TestDisplayDirectionAndTextFormatting(t *testing.T) {
	if got := displayDirection(DirectionAToB); got != "接收" {
		t.Fatalf("displayDirection(DirectionAToB) = %q", got)
	}
	if got := displayDirection(DirectionBToA); got != "发送" {
		t.Fatalf("displayDirection(DirectionBToA) = %q", got)
	}
	if got := displayDirection("custom"); got != "custom" {
		t.Fatalf("displayDirection(custom) = %q", got)
	}

	if got := normalizeEncoding(" UTF_8 "); got != "utf-8" {
		t.Fatalf("normalizeEncoding = %q", got)
	}
	if got := normalizeEncoding(" "); got != "utf-8" {
		t.Fatalf("normalizeEncoding empty = %q", got)
	}

	data := []byte{0x00, '\n', 'A', 0x7f, 0xff, 0xe4, 0xbd, 0xa0}
	if got := displayText(data); got != `\NUL\LFA\DEL�你` {
		t.Fatalf("displayText = %q", got)
	}

	frame := enrichFrame(Frame{Direction: DirectionAToB, Data: []byte{0x01, 0x0f, 0xff}}, " ASCII ")
	if frame.Encoding != "ascii" || frame.Length != 3 {
		t.Fatalf("enriched frame metadata = encoding %q length %d", frame.Encoding, frame.Length)
	}
	if frame.DisplayText != `\SOH\SI�` {
		t.Fatalf("DisplayText = %q", frame.DisplayText)
	}
	if frame.DisplayHex != "01 0f ff" || frame.DisplayDec != "1 15 255" || frame.DisplayOct != "001 017 377" || frame.DisplayBin != "00000001 00001111 11111111" {
		t.Fatalf("formatted displays = hex %q dec %q oct %q bin %q", frame.DisplayHex, frame.DisplayDec, frame.DisplayOct, frame.DisplayBin)
	}
}
