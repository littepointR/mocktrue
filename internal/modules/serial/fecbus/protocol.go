package fecbus

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	minFrameBytes = 11
	maxDataBytes  = 8
)

// EncodeFrame encodes one FECbus serial frame.
func EncodeFrame(frame Frame) ([]byte, error) {
	if err := validateFrameFields(frame); err != nil {
		return nil, err
	}
	body := []byte{
		FrameBoundary,
		byte(frame.Type),
		frame.TargetAddress,
		frame.Priority,
		frame.SourceAddress,
		frame.MessageNumber,
		frame.GroupNumber,
		byte(len(frame.Data)),
	}
	body = append(body, frame.Data...)
	crc := CRC16(body)
	body = append(body, byte(crc), byte(crc>>8), FrameBoundary)
	return body, nil
}

// DecodeFrame verifies and decodes one FECbus serial frame.
func DecodeFrame(raw []byte) (Frame, error) {
	if len(raw) < minFrameBytes {
		return Frame{}, fmt.Errorf("fecbus frame too short")
	}
	if raw[0] != FrameBoundary {
		return Frame{}, fmt.Errorf("fecbus frame must start with 0x7e")
	}
	if raw[len(raw)-1] != FrameBoundary {
		return Frame{}, fmt.Errorf("fecbus frame must end with 0x7e")
	}
	dlc := int(raw[7])
	if dlc < 1 || dlc > maxDataBytes {
		return Frame{}, fmt.Errorf("fecbus data length must be 1-8")
	}
	if len(raw) != 1+7+dlc+2+1 {
		return Frame{}, fmt.Errorf("fecbus data length mismatch")
	}
	payload := raw[:len(raw)-3]
	got := uint16(raw[len(raw)-3]) | uint16(raw[len(raw)-2])<<8
	if want := CRC16(payload); got != want {
		return Frame{}, fmt.Errorf("fecbus crc mismatch")
	}
	frame := Frame{
		Type:          FrameType(raw[1]),
		TargetAddress: raw[2],
		Priority:      raw[3],
		SourceAddress: raw[4],
		MessageNumber: raw[5],
		GroupNumber:   raw[6],
		Data:          append([]byte(nil), raw[8:8+dlc]...),
		Raw:           append([]byte(nil), raw...),
		CRCOK:         true,
		Timestamp:     time.Now(),
	}
	if err := validateFrameFields(frame); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

func validateFrameFields(frame Frame) error {
	if frame.Type != FrameTypeRequest && frame.Type != FrameTypeAnswer {
		return fmt.Errorf("fecbus frame type must be 0 or 1")
	}
	if frame.TargetAddress > 63 {
		return fmt.Errorf("fecbus target address must be 0-63")
	}
	if frame.Priority > 3 {
		return fmt.Errorf("fecbus priority must be 0-3")
	}
	if frame.SourceAddress == 0 || frame.SourceAddress > 63 {
		return fmt.Errorf("fecbus source address must be 1-63")
	}
	if frame.MessageNumber == 0 || frame.MessageNumber > 63 {
		return fmt.Errorf("fecbus message number must be 1-63")
	}
	if frame.GroupNumber > 127 {
		return fmt.Errorf("fecbus group number must be 0-127")
	}
	if len(frame.Data) < 1 || len(frame.Data) > maxDataBytes {
		return fmt.Errorf("fecbus data length must be 1-8")
	}
	if frame.GroupNumber <= 1 && !IsFunctionAllowed(FunctionCode(frame.Data[0])) {
		return fmt.Errorf("fecbus function code %d is reserved", frame.Data[0])
	}
	return nil
}

// CRC16 returns the FECbus serial CRC using polynomial 0xA001.
func CRC16(data []byte) uint16 {
	var crc uint16 = 0xffff
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xa001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// BuildStatusAnswer constructs the Appendix C status-answer frame.
func BuildStatusAnswer(request Frame, status StatusCode) (Frame, error) {
	answer := Frame{
		Type:          FrameTypeAnswer,
		TargetAddress: request.SourceAddress,
		Priority:      3,
		SourceAddress: request.TargetAddress,
		MessageNumber: request.MessageNumber,
		GroupNumber:   0,
		Data:          []byte{byte(FunctionStatusAnswer), byte(status)},
	}
	if answer.SourceAddress == 0 {
		answer.SourceAddress = 1
	}
	return answer, validateFrameFields(answer)
}

// BuildData creates a valid FECbus data payload from function and optional bytes.
func BuildData(function FunctionCode, payload []byte) ([]byte, error) {
	if !IsFunctionAllowed(function) {
		return nil, fmt.Errorf("fecbus function code %d is reserved", function)
	}
	if len(payload) > maxDataBytes-1 {
		return nil, fmt.Errorf("fecbus function payload must be 0-7 bytes")
	}
	data := make([]byte, 1, 1+len(payload))
	data[0] = byte(function)
	data = append(data, payload...)
	return data, nil
}

// ParseHex parses whitespace-separated HEX bytes.
func ParseHex(value string) ([]byte, error) {
	compact := strings.NewReplacer(" ", "", "\n", "", "\t", "", "\r", "").Replace(value)
	if compact == "" {
		return nil, nil
	}
	data, err := hex.DecodeString(compact)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// FormatHex formats bytes as lowercase, space-separated HEX.
func FormatHex(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02x", b))
	}
	return strings.Join(parts, " ")
}

// ExtractFrame returns the first complete frame from buf.
func ExtractFrame(buf []byte) (frame []byte, rest []byte, ok bool, err error) {
	start := -1
	for i, b := range buf {
		if b == FrameBoundary {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, nil, false, nil
	}
	if start > 0 {
		buf = buf[start:]
	}
	if len(buf) < 2 {
		return nil, buf, false, nil
	}
	end := -1
	for i := 1; i < len(buf); i++ {
		if buf[i] == FrameBoundary {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, buf, false, nil
	}
	candidate := append([]byte(nil), buf[:end+1]...)
	if len(candidate) < minFrameBytes {
		return nil, buf[end+1:], false, fmt.Errorf("fecbus frame too short")
	}
	return candidate, append([]byte(nil), buf[end+1:]...), true, nil
}
