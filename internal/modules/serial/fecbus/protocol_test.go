package fecbus

import (
	"encoding/hex"
	"testing"
)

func TestGoldenSyncRequestFrame(t *testing.T) {
	frame, err := EncodeFrame(Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 2,
		Priority:      0,
		SourceAddress: 1,
		MessageNumber: 1,
		GroupNumber:   0,
		Data:          []byte{byte(FunctionSyncHeartbeat)},
	})
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	assertHex(t, frame, "7e00020001010001003d3e7e")

	decoded, err := DecodeFrame(frame)
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	if decoded.Type != FrameTypeRequest {
		t.Fatalf("Type = %d, want request", decoded.Type)
	}
	if decoded.TargetAddress != 2 || decoded.SourceAddress != 1 {
		t.Fatalf("addresses = target %d source %d, want 2/1", decoded.TargetAddress, decoded.SourceAddress)
	}
	if decoded.MessageNumber != 1 || decoded.GroupNumber != 0 {
		t.Fatalf("numbers = message %d group %d, want 1/0", decoded.MessageNumber, decoded.GroupNumber)
	}
	if got := decoded.Function(); got != FunctionSyncHeartbeat {
		t.Fatalf("Function() = %d, want sync", got)
	}
}

func TestDecodeRejectsMalformedFrames(t *testing.T) {
	valid := mustHex(t, "7e00020001010001003d3e7e")

	tests := []struct {
		name  string
		frame []byte
	}{
		{"too short", []byte{0x7e, 0x00, 0x7e}},
		{"bad header", append([]byte{0x00}, valid[1:]...)},
		{"bad footer", append(append([]byte(nil), valid[:len(valid)-1]...), 0x00)},
		{"crc mismatch", corrupt(valid, 3)},
		{"dlc too large", mustHex(t, "7e00020001010009003d3e7e")},
		{"dlc mismatch", mustHex(t, "7e00020001010002003d3e7e")},
		{"invalid frame type", mustFrameWithFields(t, 0x02, 1, 0, 2, 1, 0, []byte{0})},
		{"invalid priority", mustFrameWithFields(t, 0x00, 1, 4, 2, 1, 0, []byte{0})},
		{"invalid source address", mustFrameWithFields(t, 0x00, 1, 0, 0, 1, 0, []byte{0})},
		{"invalid message number", mustFrameWithFields(t, 0x00, 1, 0, 2, 0, 0, []byte{0})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DecodeFrame(tt.frame); err == nil {
				t.Fatalf("DecodeFrame() error = nil, want error")
			}
		})
	}
}

func TestDecodeAllowsContinuationFrameWithoutFunctionCode(t *testing.T) {
	raw := mustFrameWithFields(t, 0x01, 1, 3, 2, 9, 2, []byte{0x80, 0x81, 0x82})
	frame, err := DecodeFrame(raw)
	if err != nil {
		t.Fatalf("DecodeFrame(continuation) error = %v", err)
	}
	if frame.GroupNumber != 2 || frame.Data[0] != 0x80 {
		t.Fatalf("continuation frame = %#v, want group 2 raw data", frame)
	}
}

func TestFunctionCatalogCoversTableC2(t *testing.T) {
	known := []FunctionCode{0, 1, 2, 3, 4, 5, 6, 7, 15, 17, 18, 19, 20, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45}
	for _, code := range known {
		info, ok := LookupFunction(code)
		if !ok {
			t.Fatalf("LookupFunction(%d) not found", code)
		}
		if info.Reserved || info.Custom {
			t.Fatalf("LookupFunction(%d) = %#v, want named non-reserved", code, info)
		}
	}

	for _, code := range []FunctionCode{8, 14, 21, 31, 46, 127} {
		info, ok := LookupFunction(code)
		if !ok || !info.Custom || info.Reserved {
			t.Fatalf("LookupFunction(%d) = %#v %v, want custom", code, info, ok)
		}
	}

	for _, code := range []FunctionCode{16, 32, 128, 200, 255} {
		info, ok := LookupFunction(code)
		if !ok || !info.Reserved || IsFunctionAllowed(code) {
			t.Fatalf("LookupFunction(%d) = %#v %v, want reserved disallowed", code, info, ok)
		}
	}
}

func TestBuildStatusAnswer(t *testing.T) {
	answer, err := BuildStatusAnswer(Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 2,
		SourceAddress: 1,
		MessageNumber: 9,
		GroupNumber:   7,
	}, StatusReceivedOK)
	if err != nil {
		t.Fatalf("BuildStatusAnswer() error = %v", err)
	}
	if answer.Type != FrameTypeAnswer || answer.Priority != 3 {
		t.Fatalf("answer type/priority = %d/%d, want answer/3", answer.Type, answer.Priority)
	}
	if answer.TargetAddress != 1 || answer.SourceAddress != 2 {
		t.Fatalf("answer addresses = %d/%d, want swapped 1/2", answer.TargetAddress, answer.SourceAddress)
	}
	if answer.GroupNumber != 0 {
		t.Fatalf("answer group = %d, want 0", answer.GroupNumber)
	}
	assertHex(t, answer.Data, "0f0a")
}

func TestProtocolHelpersCoverErrorAndExtractionEdges(t *testing.T) {
	if _, err := EncodeFrame(Frame{Type: FrameTypeRequest, TargetAddress: 64, Priority: 0, SourceAddress: 1, MessageNumber: 1, Data: []byte{byte(FunctionSyncHeartbeat)}}); err == nil {
		t.Fatalf("EncodeFrame(invalid target) error = nil, want error")
	}
	if _, err := BuildData(FunctionCode(0x10), nil); err == nil {
		t.Fatalf("BuildData(reserved) error = nil, want error")
	}
	if _, err := BuildData(FunctionSyncHeartbeat, []byte{1, 2, 3, 4, 5, 6, 7, 8}); err == nil {
		t.Fatalf("BuildData(oversized payload) error = nil, want error")
	}
	if data, err := ParseHex(" \n	\r "); err != nil || data != nil {
		t.Fatalf("ParseHex(whitespace) = %x/%v, want nil/nil", data, err)
	}

	valid := mustHex(t, "7e00020001010001003d3e7e")
	frame, rest, ok, err := ExtractFrame(append([]byte{0x00, 0x01}, valid...))
	if err != nil || !ok || string(frame) != string(valid) || len(rest) != 0 {
		t.Fatalf("ExtractFrame(with prefix) = frame %x rest %x ok %v err %v, want valid/no rest", frame, rest, ok, err)
	}
	frame, rest, ok, err = ExtractFrame([]byte{0x00, 0x7e, 0x7e, 0x01, 0x02})
	if err == nil || ok || frame != nil || string(rest) != string([]byte{0x01, 0x02}) {
		t.Fatalf("ExtractFrame(too short candidate) = frame %x rest %x ok %v err %v, want short error and remaining bytes", frame, rest, ok, err)
	}
	frame, rest, ok, err = ExtractFrame([]byte{0x00, 0x7e, 0x00})
	if err != nil || ok || frame != nil || string(rest) != string([]byte{0x7e, 0x00}) {
		t.Fatalf("ExtractFrame(partial) = frame %x rest %x ok %v err %v, want buffered partial", frame, rest, ok, err)
	}

	answer, err := BuildStatusAnswer(Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 0,
		SourceAddress: 4,
		MessageNumber: 2,
	}, StatusBusy)
	if err != nil {
		t.Fatalf("BuildStatusAnswer(default source) error = %v", err)
	}
	if answer.TargetAddress != 4 || answer.SourceAddress != 1 || StatusCode(answer.Data[1]) != StatusBusy {
		t.Fatalf("BuildStatusAnswer(default source) = target %d source %d data %x, want 4/1 busy", answer.TargetAddress, answer.SourceAddress, answer.Data)
	}
}

func mustFrameWithFields(t *testing.T, ft byte, da byte, pa byte, sa byte, mn byte, tn byte, data []byte) []byte {
	t.Helper()
	body := []byte{FrameBoundary, ft, da, pa, sa, mn, tn, byte(len(data))}
	body = append(body, data...)
	crc := CRC16(body)
	body = append(body, byte(crc), byte(crc>>8), FrameBoundary)
	return body
}

func corrupt(data []byte, index int) []byte {
	out := append([]byte(nil), data...)
	out[index] ^= 0xff
	return out
}

func mustHex(t *testing.T, value string) []byte {
	t.Helper()
	data, err := hex.DecodeString(value)
	if err != nil {
		t.Fatalf("hex.DecodeString(%q) error = %v", value, err)
	}
	return data
}

func assertHex(t *testing.T, got []byte, wantHex string) {
	t.Helper()
	if gotHex := hex.EncodeToString(got); gotHex != wantHex {
		t.Fatalf("hex = %s, want %s", gotHex, wantHex)
	}
}
