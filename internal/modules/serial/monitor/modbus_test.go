package monitor

import "testing"

func TestParseModbusRTU(t *testing.T) {
	frame := ParseModbus([]byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0a, 0xc5, 0xcd})
	if frame == nil {
		t.Fatalf("ParseModbus returned nil")
	}
	if frame.Protocol != "rtu" || frame.Slave != 1 || frame.Function != 3 {
		t.Fatalf("parsed frame = %+v, want rtu slave 1 function 3", frame)
	}
	if !frame.CRCOK || frame.Error != "" {
		t.Fatalf("crc parse = ok %v error %q, want ok", frame.CRCOK, frame.Error)
	}
	if frame.Summary != "站号 1 读保持寄存器 0x03" {
		t.Fatalf("summary = %q", frame.Summary)
	}
}

func TestParseModbusRTUReportsCRCErrors(t *testing.T) {
	frame := ParseModbus([]byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x00})
	if frame == nil {
		t.Fatalf("ParseModbus returned nil")
	}
	if frame.CRCOK || frame.Error == "" {
		t.Fatalf("bad crc frame = %+v, want error", frame)
	}
}

func TestParseModbusASCII(t *testing.T) {
	frame := ParseModbus([]byte(":01030000000AF2\r\n"))
	if frame == nil {
		t.Fatalf("ParseModbus returned nil")
	}
	if frame.Protocol != "ascii" || frame.Slave != 1 || frame.Function != 3 {
		t.Fatalf("parsed frame = %+v, want ascii slave 1 function 3", frame)
	}
	if !frame.LRCOK {
		t.Fatalf("LRCOK = false, frame = %+v", frame)
	}
}

func TestDisplayTextEscapesControlBytes(t *testing.T) {
	got := displayText([]byte{0x00, 0x01, 'A', '\r', '\n'})
	if got != `\NUL\SOHA\CR\LF` {
		t.Fatalf("displayText = %q", got)
	}
}
