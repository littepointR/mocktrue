package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultUTF8Registers = 24

// EncodeRegisterValue converts a frontend value into Modbus register words.
func EncodeRegisterValue(spec RegisterValueSpec, input string) ([]uint16, error) {
	dataType := normalizeDataType(spec.DataType)
	length := registerLength(spec)
	if length == 0 {
		return nil, fmt.Errorf("data type must not be none")
	}
	var bytes []byte
	switch dataType {
	case DataTypeInt16:
		value, err := strconv.ParseInt(strings.TrimSpace(input), 10, 16)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 2)
		binary.BigEndian.PutUint16(bytes, uint16(int16(value)))
	case DataTypeUint16:
		value, err := strconv.ParseUint(strings.TrimSpace(input), 10, 16)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 2)
		binary.BigEndian.PutUint16(bytes, uint16(value))
	case DataTypeInt32:
		value, err := strconv.ParseInt(strings.TrimSpace(input), 10, 32)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, uint32(int32(value)))
	case DataTypeUint32, DataTypeUnix:
		value, err := strconv.ParseUint(strings.TrimSpace(input), 10, 32)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, uint32(value))
	case DataTypeInt64:
		value, err := strconv.ParseInt(strings.TrimSpace(input), 10, 64)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, uint64(value))
	case DataTypeUint64:
		value, err := strconv.ParseUint(strings.TrimSpace(input), 10, 64)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, value)
	case DataTypeFloat:
		value, err := strconv.ParseFloat(strings.TrimSpace(input), 32)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, math.Float32bits(float32(value)))
	case DataTypeDouble:
		value, err := strconv.ParseFloat(strings.TrimSpace(input), 64)
		if err != nil {
			return nil, err
		}
		bytes = make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, math.Float64bits(value))
	case DataTypeDateTime:
		value, err := strconv.ParseInt(strings.TrimSpace(input), 10, 64)
		if err != nil {
			return nil, err
		}
		bytes = encodeIEC870DateTime(value)
	case DataTypeUTF8:
		bytes = make([]byte, int(length)*2)
		copy(bytes, []byte(input))
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
	if spec.WordOrder == WordOrderLittle {
		reverseRegisterWords(bytes)
	}
	return bytesToRegisters(bytes), nil
}

// DecodeRegisterValue converts Modbus register words into a display value.
func DecodeRegisterValue(spec RegisterValueSpec, registers []uint16) (DecodedRegisterValue, error) {
	dataType := normalizeDataType(spec.DataType)
	length := int(registerLength(spec))
	if length == 0 {
		return DecodedRegisterValue{}, fmt.Errorf("data type must not be none")
	}
	if len(registers) < length {
		return DecodedRegisterValue{}, fmt.Errorf("need %d registers, got %d", length, len(registers))
	}
	raw := append([]uint16(nil), registers[:length]...)
	bytes := registersToBytes(raw)
	if spec.WordOrder == WordOrderLittle {
		reverseRegisterWords(bytes)
	}

	out := DecodedRegisterValue{DataType: dataType, Raw: raw}
	switch dataType {
	case DataTypeInt16:
		out.Numeric = float64(int16(binary.BigEndian.Uint16(bytes)))
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeUint16:
		out.Numeric = float64(binary.BigEndian.Uint16(bytes))
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeInt32:
		out.Numeric = float64(int32(binary.BigEndian.Uint32(bytes)))
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeUint32:
		out.Numeric = float64(binary.BigEndian.Uint32(bytes))
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeInt64:
		value := int64(binary.BigEndian.Uint64(bytes))
		out.Numeric = float64(value)
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeUint64:
		value := binary.BigEndian.Uint64(bytes)
		out.Numeric = float64(value)
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeFloat:
		out.Numeric = float64(math.Float32frombits(binary.BigEndian.Uint32(bytes)))
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeDouble:
		out.Numeric = math.Float64frombits(binary.BigEndian.Uint64(bytes))
		out.Display = formatNumber(applyNumericTransforms(out.Numeric, spec))
	case DataTypeUnix:
		seconds := int64(binary.BigEndian.Uint32(bytes))
		out.Numeric = float64(seconds)
		out.Display = formatModbusTime(time.Unix(seconds, 0), spec.LocalTime)
	case DataTypeDateTime:
		display, err := decodeIEC870DateTime(bytes, spec.LocalTime)
		if err != nil {
			return DecodedRegisterValue{}, err
		}
		out.Display = display
	case DataTypeUTF8:
		out.Display = strings.TrimRight(string(trimInvalidUTF8(bytes)), "\x00 ")
	default:
		return DecodedRegisterValue{}, fmt.Errorf("unsupported data type: %s", dataType)
	}
	if dataType != DataTypeUnix && dataType != DataTypeDateTime && dataType != DataTypeUTF8 {
		out.Numeric = applyNumericTransforms(out.Numeric, spec)
	}
	return out, nil
}

// GroupRegisterMappings builds continuous read blocks from configured registers.
func GroupRegisterMappings(mappings []RegisterMapping, maxLength uint16) []AddressGroup {
	if maxLength == 0 {
		maxLength = 100
	}
	items := append([]RegisterMapping(nil), mappings...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Address < items[j].Address
	})

	groups := make([]AddressGroup, 0)
	for i := 0; i < len(items); {
		if normalizeDataType(items[i].DataType) == DataTypeNone {
			i++
			continue
		}
		start := items[i].Address
		end := mappingEnd(items[i])
		j := i
		for j+1 < len(items) {
			if items[j].GroupEnd {
				break
			}
			next := items[j+1]
			if normalizeDataType(next.DataType) == DataTypeNone {
				j++
				continue
			}
			nextEnd := mappingEnd(next)
			candidateEnd := end
			if nextEnd > candidateEnd {
				candidateEnd = nextEnd
			}
			if candidateEnd-start+1 > maxLength {
				break
			}
			end = candidateEnd
			j++
		}
		groups = append(groups, AddressGroup{Address: start, Quantity: end - start + 1})
		i = j + 1
	}
	return groups
}

func mappingEnd(mapping RegisterMapping) uint16 {
	spec := RegisterValueSpec{DataType: mapping.DataType, Length: mapping.Length}
	length := registerLength(spec)
	if length == 0 {
		length = 1
	}
	return mapping.Address + length - 1
}

func registerLength(spec RegisterValueSpec) uint16 {
	switch normalizeDataType(spec.DataType) {
	case DataTypeNone:
		return 0
	case DataTypeInt16, DataTypeUint16:
		return 1
	case DataTypeInt32, DataTypeUint32, DataTypeFloat, DataTypeUnix:
		return 2
	case DataTypeInt64, DataTypeUint64, DataTypeDouble, DataTypeDateTime:
		return 4
	case DataTypeUTF8:
		if spec.Length > 0 {
			return spec.Length
		}
		return defaultUTF8Registers
	default:
		return 0
	}
}

func normalizeDataType(dataType DataType) DataType {
	if dataType == "" {
		return DataTypeUint16
	}
	return DataType(strings.ToLower(string(dataType)))
}

func bytesToRegisters(data []byte) []uint16 {
	out := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		out = append(out, binary.BigEndian.Uint16(data[i:i+2]))
	}
	return out
}

func registersToBytes(registers []uint16) []byte {
	out := make([]byte, len(registers)*2)
	for i, value := range registers {
		binary.BigEndian.PutUint16(out[i*2:i*2+2], value)
	}
	return out
}

func reverseRegisterWords(data []byte) {
	for left, right := 0, len(data)-2; left < right; left, right = left+2, right-2 {
		data[left], data[right] = data[right], data[left]
		data[left+1], data[right+1] = data[right+1], data[left+1]
	}
}

func encodeIEC870DateTime(timestampMs int64) []byte {
	if timestampMs < 946684800000 {
		timestampMs = 946684800000
	}
	dt := time.UnixMilli(timestampMs).UTC()
	out := make([]byte, 8)
	binary.BigEndian.PutUint16(out[0:2], uint16(dt.Year()-2000))
	binary.BigEndian.PutUint16(out[2:4], uint16(dt.Month())<<8|uint16(dt.Day()))
	binary.BigEndian.PutUint16(out[4:6], uint16(dt.Hour())<<8|uint16(dt.Minute()))
	binary.BigEndian.PutUint16(out[6:8], uint16(dt.Second()*1000+dt.Nanosecond()/1_000_000))
	return out
}

func decodeIEC870DateTime(data []byte, localTime bool) (string, error) {
	if len(data) != 8 {
		return "", fmt.Errorf("datetime requires 4 registers")
	}
	word1 := binary.BigEndian.Uint16(data[0:2])
	word2 := binary.BigEndian.Uint16(data[2:4])
	word3 := binary.BigEndian.Uint16(data[4:6])
	word4 := binary.BigEndian.Uint16(data[6:8])
	if word1 == 0xffff && word2 == 0xffff && word3 == 0xffff && word4 == 0xffff {
		return "", nil
	}
	year := int(word1&0x7f) + 2000
	month := time.Month((word2 >> 8) & 0x0f)
	day := int(word2 & 0x1f)
	hour := int((word3 >> 8) & 0x1f)
	minute := int(word3 & 0x3f)
	totalMs := int(word4)
	second := totalMs / 1000
	millisecond := totalMs % 1000
	invalid := word3&0x80 != 0
	if invalid || year < 2000 || year > 2127 || month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 || millisecond > 999 {
		return "", nil
	}
	return formatModbusTime(time.Date(year, month, day, hour, minute, second, millisecond*1_000_000, time.UTC), localTime), nil
}

func formatModbusTime(t time.Time, localTime bool) string {
	if localTime {
		return t.Local().Format("2006/01/02 15:04:05")
	}
	return t.UTC().Format("2006/01/02 15:04:05")
}

func applyNumericTransforms(value float64, spec RegisterValueSpec) float64 {
	if spec.ScalingFactor != 0 {
		value *= spec.ScalingFactor
	}
	if spec.Interpolate != nil {
		value = linearInterpolate(value, *spec.Interpolate)
	}
	return value
}

func linearInterpolate(value float64, interpolation LinearInterpolation) float64 {
	if interpolation.X2 == interpolation.X1 {
		return interpolation.Y1
	}
	t := (value - interpolation.X1) / (interpolation.X2 - interpolation.X1)
	return interpolation.Y1 + t*(interpolation.Y2-interpolation.Y1)
}

func formatNumber(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	if math.Trunc(value) == value {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func trimInvalidUTF8(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}
	return []byte(strings.ToValidUTF8(string(data), ""))
}
