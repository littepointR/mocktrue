package monitor

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

var asciiControlNames = [...]string{
	"NUL", "SOH", "STX", "ETX", "EOT", "ENQ", "ACK", "BEL",
	"BS", "HT", "LF", "VT", "FF", "CR", "SO", "SI",
	"DLE", "DC1", "DC2", "DC3", "DC4", "NAK", "SYN", "ETB",
	"CAN", "EM", "SUB", "ESC", "FS", "GS", "RS", "US",
}

func enrichFrame(frame Frame, encoding string) Frame {
	frame.Encoding = normalizeEncoding(encoding)
	frame.Length = len(frame.Data)
	frame.DisplayText = displayText(frame.Data)
	frame.DisplayHex = joinBytes(frame.Data, 16, "%02x")
	frame.DisplayDec = joinBytes(frame.Data, 10, "%d")
	frame.DisplayOct = joinBytes(frame.Data, 8, "%03o")
	frame.DisplayBin = joinBytes(frame.Data, 2, "%08b")
	return frame
}

func displayDirection(direction string) string {
	switch direction {
	case DirectionAToB:
		return "接收"
	case DirectionBToA:
		return "发送"
	default:
		return direction
	}
}

func normalizeEncoding(encoding string) string {
	encoding = strings.TrimSpace(strings.ToLower(strings.ReplaceAll(encoding, "_", "-")))
	if encoding == "" {
		return "utf-8"
	}
	return encoding
}

func displayText(data []byte) string {
	var b strings.Builder
	for len(data) > 0 {
		c := data[0]
		if c < 0x20 {
			b.WriteByte('\\')
			b.WriteString(asciiControlNames[c])
			data = data[1:]
			continue
		}
		if c == 0x7f {
			b.WriteString("\\DEL")
			data = data[1:]
			continue
		}
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			b.WriteRune('�')
			data = data[1:]
			continue
		}
		b.WriteRune(r)
		data = data[size:]
	}
	return b.String()
}

func joinBytes(data []byte, base int, format string) string {
	_ = base
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf(format, b))
	}
	return strings.Join(parts, " ")
}
