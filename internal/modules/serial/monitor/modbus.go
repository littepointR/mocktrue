package monitor

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// ParseModbus returns a best-effort parse for Modbus RTU or ASCII frames.
func ParseModbus(data []byte) *ModbusFrame {
	if len(data) == 0 {
		return nil
	}
	if data[0] == ':' {
		return parseModbusASCII(data)
	}
	return parseModbusRTU(data)
}

func parseModbusRTU(data []byte) *ModbusFrame {
	if len(data) < 4 {
		return nil
	}
	payload := data[:len(data)-2]
	got := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	want := modbusCRC(payload)
	result := &ModbusFrame{
		Protocol:    "rtu",
		Slave:       payload[0],
		Function:    payload[1],
		FunctionHex: fmt.Sprintf("0x%02x", payload[1]),
		PayloadHex:  joinBytes(payload[2:], 16, "%02x"),
		CRCOK:       got == want,
	}
	result.Summary = modbusSummary(result)
	if !result.CRCOK {
		result.Error = fmt.Sprintf("crc mismatch: got 0x%04x want 0x%04x", got, want)
	}
	return result
}

func parseModbusASCII(data []byte) *ModbusFrame {
	text := strings.TrimSpace(string(data))
	if len(text) < 9 || text[0] != ':' {
		return nil
	}
	raw, err := hex.DecodeString(text[1:])
	if err != nil || len(raw) < 3 {
		return nil
	}
	payload := raw[:len(raw)-1]
	got := raw[len(raw)-1]
	want := modbusLRC(payload)
	result := &ModbusFrame{
		Protocol:    "ascii",
		Slave:       payload[0],
		Function:    payload[1],
		FunctionHex: fmt.Sprintf("0x%02x", payload[1]),
		PayloadHex:  joinBytes(payload[2:], 16, "%02x"),
		LRCOK:       got == want,
	}
	result.Summary = modbusSummary(result)
	if !result.LRCOK {
		result.Error = fmt.Sprintf("lrc mismatch: got 0x%02x want 0x%02x", got, want)
	}
	return result
}

func modbusSummary(frame *ModbusFrame) string {
	name := "功能码"
	switch frame.Function {
	case 1:
		name = "读线圈"
	case 2:
		name = "读离散输入"
	case 3:
		name = "读保持寄存器"
	case 4:
		name = "读输入寄存器"
	case 5:
		name = "写单个线圈"
	case 6:
		name = "写单个寄存器"
	case 15:
		name = "写多个线圈"
	case 16:
		name = "写多个寄存器"
	}
	return fmt.Sprintf("站号 %d %s %s", frame.Slave, name, frame.FunctionHex)
}

func modbusCRC(data []byte) uint16 {
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

func modbusLRC(data []byte) byte {
	var sum byte
	for _, b := range data {
		sum += b
	}
	return byte(-int8(sum))
}
