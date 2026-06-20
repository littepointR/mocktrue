package fecbus

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

const groupPaletteSize = 8

var statusMeanings = map[byte]string{
	byte(StatusGroupEnd):            "分组结束",
	byte(StatusCRCError):            "CRC 校验错",
	byte(StatusInvalidService):      "无效服务",
	byte(StatusUnitFault):           "单元故障",
	byte(StatusBusy):                "忙",
	byte(StatusUnrecognizedCommand): "命令不识别",
	byte(StatusAddressNotFound):     "地址不存在",
	byte(StatusParameterError):      "参数错误",
	byte(StatusProcessing):          "处理中",
	byte(StatusEventEnd):            "事件结束",
	byte(StatusReceivedOK):          "接收正确",
}

var deviceTypeMeanings = map[uint16]string{
	1:     "控制器/具有联动控制功能的控制器",
	2:     "消防联动控制器",
	10:    "火灾显示盘",
	20:    "火灾探测器",
	30:    "感温火灾探测器",
	40:    "火焰探测器",
	61:    "手动火灾报警按钮",
	81:    "火灾警报装置",
	100:   "可燃气体报警控制器",
	110:   "电气火灾监控设备",
	120:   "应急照明控制器",
	130:   "防火门监控器",
	140:   "消防设备电源监控器",
	150:   "气体灭火控制器",
	163:   "消防电气控制装置",
	179:   "消防泵控制器",
	65535: "连接了多种设备",
}

var eventTypeMeanings = map[uint16]string{
	1:   "正常",
	2:   "首火警",
	3:   "火警",
	4:   "电气火灾报警",
	19:  "启动",
	29:  "停止",
	40:  "应急",
	70:  "监管",
	72:  "屏蔽",
	80:  "故障",
	100: "故障恢复",
	120: "开机",
	121: "关机",
	122: "复位",
	123: "自检",
	127: "消音",
}

var parameterTypeMeanings = map[byte]string{
	1: "高度",
}

var appendixCFields = map[FunctionCode][]FieldDefinition{
	FunctionReset:                controllerFields(),
	FunctionSilence:              controllerFields(),
	FunctionSelfTest:             controllerFields(),
	FunctionPollConnectionStatus: controllerFields(),
	FunctionStopEventQuery:       controllerFields(),
	FunctionQueryProtocolVersion: controllerFields(),
	FunctionBroadcastClock: {
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("year", "年", 2, 1, "uint8"),
		field("month", "月", 3, 1, "uint8"),
		field("day", "日", 4, 1, "uint8"),
		field("hour", "时", 5, 1, "uint8"),
		field("minute", "分", 6, 1, "uint8"),
		field("second", "秒", 7, 1, "uint8"),
	},
	FunctionNotifyUrgentEvent:  eventFields(),
	FunctionNotifyGeneralEvent: eventFields(),
	FunctionNotifyDebugEvent:   eventFields(),
	FunctionDeviceUrgentEvent:  eventFields(),
	FunctionDeviceGeneralEvent: eventFields(),
	FunctionDeviceDebugEvent:   eventFields(),
	FunctionStatusAnswer: {
		{Key: "status", Label: "状态应答码", Offset: 1, Length: 1, Type: "uint8", Enum: statusMeanings},
	},
	FunctionQueryDeviceStatus: scopedIDFields(),
	FunctionQueryConfig:       scopedIDFields(),
	FunctionQueryIdentifier:   scopedIDFields(),
	FunctionQueryParameter:    append(scopedIDFields(), field("parameter_type", "参量类型代码", 5, 1, "uint8").withByteEnum(parameterTypeMeanings)),
	FunctionQueryRegistration: scopedIDFields(),
	FunctionQueryCurrentEvent: []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("event_code", "事件代码", 2, 2, "uint16le").withUint16Enum(eventTypeMeanings),
	},
	FunctionQueryHistoryEvent: []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("event_code", "事件代码", 2, 2, "uint16le").withUint16Enum(eventTypeMeanings),
		field("start_year", "起始年", 4, 1, "uint8"),
		field("start_month", "起始月", 5, 1, "uint8"),
		field("start_day", "起始日", 6, 1, "uint8"),
	},
	FunctionQueryComment: []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("unit_id", "单元编号", 2, 1, "uint8"),
		field("device_id", "设备编号", 3, 1, "uint8"),
	},
	FunctionQueryProgramming: []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("unit_id", "单元编号", 2, 1, "uint8"),
		field("device_id", "设备编号", 3, 1, "uint8"),
	},
	FunctionQueryDeviceList:   scopedIDFields(),
	FunctionHeartbeatPollInfo: {},
}

// AnnotateFrame returns segment and field metadata for one captured frame.
func AnnotateFrame(record FrameRecord, custom []CustomFunctionDefinition, context *AnnotationContext) AnnotatedFrame {
	frame := record.Frame
	raw := frame.Raw
	if len(raw) == 0 && record.Hex != "" {
		raw, _ = ParseHex(record.Hex)
	}
	functionCode := frame.Function()
	if context != nil && frame.GroupNumber > 1 && !looksLikeFunctionCode(functionCode) {
		functionCode = context.Function
	}
	function := functionInfo(functionCode, custom)
	segments := serialSegments(raw, frame)
	fields := dataAnnotations(raw, frame, function, custom, functionCode)
	groupKey := multiFrameGroupKey(record)
	annotated := AnnotatedFrame{
		Segments:        segments,
		DataFields:      fields,
		Function:        function,
		GroupKey:        groupKey,
		GroupColorIndex: groupColorIndex(groupKey),
		Summary:         frameSummary(frame, function),
	}
	if len(raw) == 0 {
		annotated.Warnings = append(annotated.Warnings, "缺少原始帧字节")
	}
	return annotated
}

// BuildDataFromFields builds a FECbus data payload from structured field values.
func BuildDataFromFields(function FunctionCode, values map[string]any, custom []CustomFunctionDefinition) ([]byte, error) {
	defs := fieldDefinitions(function, custom)
	maxLen := 1
	for _, def := range defs {
		if _, ok := values[def.Key]; !ok {
			continue
		}
		if end := def.Offset + def.Length; end > maxLen {
			maxLen = end
		}
	}
	if maxLen > maxDataBytes {
		return nil, fmt.Errorf("fecbus structured payload is too long")
	}
	data := make([]byte, maxLen)
	data[0] = byte(function)
	for _, def := range defs {
		value, ok := values[def.Key]
		if !ok {
			continue
		}
		if err := encodeFieldValue(data, def.Offset, def.Length, def.Type, def.Endian, value); err != nil {
			return nil, fmt.Errorf("encode %s: %w", def.Key, err)
		}
	}
	return data, nil
}

func serialSegments(raw []byte, frame Frame) []FrameSegment {
	if len(raw) == 0 {
		return nil
	}
	segments := []FrameSegment{
		segment("frame_head", "帧头", 0, 1, raw, "0x7E"),
		segment("frame_type", "帧类型 FT", 1, 2, raw, frameTypeMeaning(frame.Type)),
		segment("target_address", "目标地址 DA", 2, 3, raw, fmt.Sprintf("%d", frame.TargetAddress)),
		segment("priority", "优先级 PA", 3, 4, raw, fmt.Sprintf("%d", frame.Priority)),
		segment("source_address", "源地址 SA", 4, 5, raw, fmt.Sprintf("%d", frame.SourceAddress)),
		segment("message_number", "报文编号 MN", 5, 6, raw, fmt.Sprintf("%d", frame.MessageNumber)),
		segment("group_number", "分组编号 TN", 6, 7, raw, groupNumberMeaning(frame.GroupNumber)),
		segment("data_length", "数据长度 DLC", 7, 8, raw, fmt.Sprintf("%d", len(frame.Data))),
	}
	dataEnd := 8 + len(frame.Data)
	if dataEnd <= len(raw)-3 {
		segments = append(segments, segment("data", "报文 Data", 8, dataEnd, raw, "应用数据单元"))
		segments = append(segments, segment("crc", "CRC0/CRC1", len(raw)-3, len(raw)-1, raw, crcMeaning(frame.CRCOK)))
		segments = append(segments, segment("frame_tail", "帧尾", len(raw)-1, len(raw), raw, "0x7E"))
	}
	return segments
}

func dataAnnotations(raw []byte, frame Frame, info FunctionInfo, custom []CustomFunctionDefinition, functionCode FunctionCode) []FieldAnnotation {
	if len(frame.Data) == 0 {
		return nil
	}
	fields := []FieldAnnotation{dataField("function", "功能码", 0, 1, raw, frame.Data, info.Name, byte(functionCode), info.Name)}
	if frame.GroupNumber > 1 && !looksLikeFunctionCode(frame.Function()) {
		return []FieldAnnotation{dataField("continuation", "续帧数据", 0, len(frame.Data), raw, frame.Data, FormatHex(frame.Data), FormatHex(frame.Data), "多帧报文后续数据")}
	}
	for _, def := range fieldDefinitionsForFrame(frame, functionCode, custom) {
		if def.Offset < 1 || def.Length <= 0 || def.Offset >= len(frame.Data) {
			continue
		}
		end := def.Offset + def.Length
		if end > len(frame.Data) {
			end = len(frame.Data)
		}
		bytes := frame.Data[def.Offset:end]
		value := decodeFieldValue(bytes, def.Type, def.Endian)
		valueText := fmt.Sprint(value)
		meaning := def.Meaning
		if len(bytes) == 1 && def.Enum != nil {
			if text, ok := def.Enum[bytes[0]]; ok {
				valueText = text
				meaning = text
			}
		}
		if len(bytes) == 2 && def.Enum16 != nil {
			rawValue := uint16(0)
			if def.Type == "uint16be" || def.Endian == "big" {
				rawValue = binary.BigEndian.Uint16(bytes[:2])
			} else {
				rawValue = binary.LittleEndian.Uint16(bytes[:2])
			}
			if text, ok := def.Enum16[rawValue]; ok {
				valueText = text
				meaning = text
			}
		}
		fields = append(fields, dataField(def.Key, def.Label, def.Offset, end, raw, frame.Data, valueText, value, meaning))
	}
	return fields
}

func dataField(key string, label string, dataStart int, dataEnd int, raw []byte, data []byte, valueText string, value any, meaning string) FieldAnnotation {
	start := 8 + dataStart
	end := 8 + dataEnd
	hexValue := ""
	if dataStart >= 0 && dataStart < len(data) {
		if dataEnd > len(data) {
			dataEnd = len(data)
		}
		hexValue = FormatHex(data[dataStart:dataEnd])
	}
	if len(raw) > 0 && end > len(raw) {
		end = len(raw)
	}
	return FieldAnnotation{
		Key:       key,
		Label:     label,
		Start:     start,
		End:       end,
		Hex:       hexValue,
		Value:     value,
		ValueText: valueText,
		Meaning:   meaning,
	}
}

func fieldDefinitions(function FunctionCode, custom []CustomFunctionDefinition) []FieldDefinition {
	for _, def := range custom {
		if def.Code != function {
			continue
		}
		out := make([]FieldDefinition, 0, len(def.Fields))
		for _, field := range def.Fields {
			out = append(out, FieldDefinition{
				Key:     field.Key,
				Label:   field.Label,
				Offset:  field.Offset,
				Length:  field.Length,
				Type:    field.Type,
				Endian:  field.Endian,
				Enum:    field.Enum,
				Meaning: field.Meaning,
			})
		}
		return out
	}
	return appendixCFields[function]
}

func fieldDefinitionsForFrame(frame Frame, function FunctionCode, custom []CustomFunctionDefinition) []FieldDefinition {
	if customFields := fieldDefinitions(function, custom); len(customFields) > 0 && hasCustomDefinition(function, custom) {
		return customFields
	}
	if frame.Type == FrameTypeAnswer {
		switch function {
		case FunctionQueryDeviceStatus:
			return []FieldDefinition{field("status_code", "状态代码", 1, 2, "uint16le").withUint16Enum(statusBitMeanings())}
		case FunctionQueryConfig:
			return []FieldDefinition{field("config_code", "配置代码", 1, 2, "uint16le")}
		case FunctionQueryIdentifier:
			return []FieldDefinition{field("identifier_code", "标识代码", 1, 7, "hex")}
		case FunctionQueryParameter:
			return []FieldDefinition{
				field("parameter_type", "参量类型代码", 1, 1, "uint8").withByteEnum(parameterTypeMeanings),
				field("parameter_value", "参量数值", 2, 2, "uint16le"),
			}
		case FunctionQueryProtocolVersion:
			return []FieldDefinition{field("version_major", "协议版本号0", 1, 1, "uint8"), field("version_minor", "协议版本号1", 2, 1, "uint8")}
		case FunctionQueryComment:
			return textAnswerFields("comment", "注释")
		case FunctionQueryProgramming:
			return textAnswerFields("programming", "编程")
		case FunctionQueryRegistration:
			return textAnswerFields("registration", "注册")
		case FunctionQueryCurrentEvent, FunctionQueryHistoryEvent:
			return eventAnswerFields()
		case FunctionQueryDeviceList:
			return deviceListAnswerFields()
		}
	}
	return fieldDefinitions(function, custom)
}

func hasCustomDefinition(function FunctionCode, custom []CustomFunctionDefinition) bool {
	for _, def := range custom {
		if def.Code == function {
			return true
		}
	}
	return false
}

func functionInfo(code FunctionCode, custom []CustomFunctionDefinition) FunctionInfo {
	for _, def := range custom {
		if def.Code == code {
			info, _ := LookupFunction(code)
			info.Name = def.Name
			info.Description = def.Description
			info.Direction = def.Direction
			info.Answer = def.Answer
			info.Custom = true
			info.Reserved = false
			return info
		}
	}
	info, _ := LookupFunction(code)
	return info
}

func multiFrameGroupKey(record FrameRecord) string {
	frame := record.Frame
	if frame.GroupNumber == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%s:%d:%d:%d", record.SessionID, record.Direction, frame.TargetAddress, frame.SourceAddress, frame.MessageNumber)
}

func groupColorIndex(key string) int {
	if key == "" {
		return -1
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(key))
	return int(hash.Sum32() % groupPaletteSize)
}

func segment(key string, label string, start int, end int, raw []byte, meaning string) FrameSegment {
	hexValue := ""
	var value any
	if start >= 0 && end <= len(raw) && start < end {
		bytes := raw[start:end]
		hexValue = FormatHex(bytes)
		if len(bytes) == 1 {
			value = bytes[0]
		} else {
			value = append([]byte(nil), bytes...)
		}
	}
	return FrameSegment{Key: key, Label: label, Start: start, End: end, Hex: hexValue, Value: value, ValueText: fmt.Sprint(value), Meaning: meaning}
}

func frameTypeMeaning(frameType FrameType) string {
	if frameType == FrameTypeAnswer {
		return "应答帧"
	}
	return "请求帧"
}

func groupNumberMeaning(group byte) string {
	if group == 0 {
		return "单帧报文"
	}
	return fmt.Sprintf("多帧报文分组 %d", group)
}

func crcMeaning(ok bool) string {
	if ok {
		return "CRC 正确"
	}
	return "CRC 未验证"
}

func frameSummary(frame Frame, info FunctionInfo) string {
	return strings.TrimSpace(fmt.Sprintf("%s DA=%d SA=%d MN=%d TN=%d", info.Name, frame.TargetAddress, frame.SourceAddress, frame.MessageNumber, frame.GroupNumber))
}

func looksLikeFunctionCode(code FunctionCode) bool {
	return IsFunctionAllowed(code)
}

func controllerFields() []FieldDefinition {
	return []FieldDefinition{field("controller_id", "控制器编号", 1, 1, "uint8")}
}

func scopedIDFields() []FieldDefinition {
	return []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("unit_id", "单元编号", 2, 1, "uint8"),
		field("device_id", "设备编号", 3, 1, "uint8"),
		field("channel_id", "通道编号", 4, 1, "uint8"),
	}
}

func eventFields() []FieldDefinition {
	return []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("unit_id", "单元编号", 2, 1, "uint8"),
		field("device_id", "设备编号", 3, 1, "uint8"),
		field("channel_id", "通道编号", 4, 1, "uint8"),
		field("device_type", "设备类型代码", 5, 2, "uint16le"),
		field("event_code", "事件代码", 7, 2, "uint16le"),
		field("status_code", "状态代码", 9, 2, "uint16le"),
		field("year", "年", 11, 1, "uint8"),
		field("month", "月", 12, 1, "uint8"),
		field("day", "日", 13, 1, "uint8"),
		field("hour", "时", 14, 1, "uint8"),
		field("minute", "分", 15, 1, "uint8"),
		field("second", "秒", 16, 1, "uint8"),
	}
}

func field(key string, label string, offset int, length int, valueType string) FieldDefinition {
	return FieldDefinition{Key: key, Label: label, Offset: offset, Length: length, Type: valueType, Endian: "little"}
}

func textAnswerFields(key string, label string) []FieldDefinition {
	return []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("unit_id", "单元编号", 2, 1, "uint8"),
		field("device_id", "设备编号", 3, 1, "uint8"),
		field(key+"_byte_count", label+"字节数", 4, 1, "uint8"),
		field(key+"_text", label+"内容", 5, 3, "utf8"),
	}
}

func eventAnswerFields() []FieldDefinition {
	return []FieldDefinition{
		field("controller_id", "控制器编号", 1, 1, "uint8"),
		field("unit_id", "单元编号", 2, 1, "uint8"),
		field("device_id", "设备编号", 3, 1, "uint8"),
		field("channel_id", "通道编号", 4, 1, "uint8"),
		field("device_type", "设备类型代码", 5, 2, "uint16le").withUint16Enum(deviceTypeMeanings),
		field("event_code", "事件代码", 7, 2, "uint16le").withUint16Enum(eventTypeMeanings),
		field("status_code", "状态代码", 9, 2, "uint16le").withUint16Enum(statusBitMeanings()),
		field("year", "年", 11, 1, "uint8"),
		field("month", "月", 12, 1, "uint8"),
		field("day", "日", 13, 1, "uint8"),
		field("hour", "时", 14, 1, "uint8"),
		field("minute", "分", 15, 1, "uint8"),
		field("second", "秒", 16, 1, "uint8"),
	}
}

func deviceListAnswerFields() []FieldDefinition {
	return []FieldDefinition{
		field("controller_count", "控制器数量", 1, 1, "uint8"),
		field("controller_1_id", "控制器编号1", 2, 1, "uint8"),
		field("controller_1_device_type", "设备类型代码1", 3, 2, "uint16le").withUint16Enum(deviceTypeMeanings),
		field("controller_2_id", "控制器编号2", 5, 1, "uint8"),
		field("controller_2_device_type", "设备类型代码2", 6, 2, "uint16le").withUint16Enum(deviceTypeMeanings),
	}
}

func statusBitMeanings() map[uint16]string {
	return map[uint16]string{
		0:       "无状态位",
		1 << 0:  "自动状态",
		1 << 1:  "备电工作",
		1 << 2:  "电源故障",
		1 << 3:  "有报警",
		1 << 4:  "有启动",
		1 << 5:  "有反馈",
		1 << 6:  "有监管",
		1 << 7:  "有故障",
		1 << 8:  "有屏蔽",
		1 << 9:  "有气体喷洒",
		1 << 10: "有应急",
	}
}

func (def FieldDefinition) withByteEnum(enum map[byte]string) FieldDefinition {
	def.Enum = enum
	return def
}

func (def FieldDefinition) withUint16Enum(enum map[uint16]string) FieldDefinition {
	def.Enum16 = enum
	return def
}

func decodeFieldValue(bytes []byte, valueType string, endian string) any {
	if len(bytes) == 0 {
		return ""
	}
	switch valueType {
	case "uint16", "uint16le", "uint16be":
		if len(bytes) < 2 {
			return int(bytes[0])
		}
		if valueType == "uint16be" || endian == "big" {
			return int(binary.BigEndian.Uint16(bytes[:2]))
		}
		return int(binary.LittleEndian.Uint16(bytes[:2]))
	case "string", "utf8":
		return string(bytes)
	default:
		if len(bytes) == 1 {
			return int(bytes[0])
		}
		return FormatHex(bytes)
	}
}

func encodeFieldValue(data []byte, offset int, length int, valueType string, endian string, value any) error {
	if offset < 1 || length <= 0 || offset+length > len(data) {
		return fmt.Errorf("field range out of payload")
	}
	target := data[offset : offset+length]
	switch valueType {
	case "string", "utf8":
		text := fmt.Sprint(value)
		copy(target, []byte(text))
		return nil
	case "hex":
		bytes, err := ParseHex(fmt.Sprint(value))
		if err != nil {
			return err
		}
		copy(target, bytes)
		return nil
	default:
		number, err := numericValue(value)
		if err != nil {
			return err
		}
		if length == 1 {
			target[0] = byte(number)
			return nil
		}
		if length == 2 {
			if valueType == "uint16be" || endian == "big" {
				binary.BigEndian.PutUint16(target, uint16(number))
			} else {
				binary.LittleEndian.PutUint16(target, uint16(number))
			}
			return nil
		}
		for i := 0; i < length; i++ {
			target[i] = byte(number >> (8 * i))
		}
		return nil
	}
}

func numericValue(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case byte:
		return int64(v), nil
	case string:
		if strings.HasPrefix(strings.ToLower(v), "0x") {
			return strconv.ParseInt(v[2:], 16, 64)
		}
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported value type %T", value)
	}
}
