package fecbus

import (
	"strings"
	"testing"
)

func TestAnnotateFrameMarksSerialSegmentsAndKnownFields(t *testing.T) {
	raw, frame := mustEncodedFrame(t, Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 2,
		Priority:      2,
		SourceAddress: 1,
		MessageNumber: 9,
		GroupNumber:   0,
		Data:          []byte{byte(FunctionQueryDeviceStatus), 0x03, 0x04},
	})

	annotated := AnnotateFrame(FrameRecord{SessionID: "s1", Direction: "tx", Frame: frame, Hex: FormatHex(raw)}, nil, nil)
	if annotated.Function.Code != FunctionQueryDeviceStatus {
		t.Fatalf("function = %d, want query status", annotated.Function.Code)
	}
	if annotated.GroupKey != "" {
		t.Fatalf("single-frame group key = %q, want empty", annotated.GroupKey)
	}
	assertSegment(t, annotated.Segments, "frame_head", 0, 1, "帧头")
	assertSegment(t, annotated.Segments, "frame_type", 1, 2, "请求帧")
	assertSegment(t, annotated.Segments, "target_address", 2, 3, "目标地址")
	assertSegment(t, annotated.Segments, "crc", len(raw)-3, len(raw)-1, "CRC")
	assertSegment(t, annotated.Segments, "frame_tail", len(raw)-1, len(raw), "帧尾")

	assertField(t, annotated.DataFields, "function", 8, 9, "查设备状态")
	assertField(t, annotated.DataFields, "controller_id", 9, 10, "控制器编号")
	assertField(t, annotated.DataFields, "unit_id", 10, 11, "单元编号")
}

func TestAnnotateFrameGroupsMultiFrameByAddressesMessageAndGroup(t *testing.T) {
	_, first := mustEncodedFrame(t, Frame{
		Type:          FrameTypeAnswer,
		TargetAddress: 1,
		Priority:      3,
		SourceAddress: 2,
		MessageNumber: 9,
		GroupNumber:   1,
		Data:          []byte{byte(FunctionQueryCurrentEvent), 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
	})
	_, second := mustEncodedFrame(t, Frame{
		Type:          FrameTypeAnswer,
		TargetAddress: 1,
		Priority:      3,
		SourceAddress: 2,
		MessageNumber: 9,
		GroupNumber:   2,
		Data:          []byte{0x08, 0x09, 0x0a, 0x18, 0x06, 0x14, 0x10, 0x30},
	})

	a := AnnotateFrame(FrameRecord{SessionID: "s1", Direction: "rx", Frame: first}, nil, nil)
	b := AnnotateFrame(FrameRecord{SessionID: "s1", Direction: "rx", Frame: second}, nil, &AnnotationContext{Function: FunctionQueryCurrentEvent})
	if a.GroupKey == "" {
		t.Fatalf("multi-frame group key is empty")
	}
	if a.GroupKey != b.GroupKey {
		t.Fatalf("group keys = %q/%q, want same", a.GroupKey, b.GroupKey)
	}
	if a.GroupColorIndex != b.GroupColorIndex {
		t.Fatalf("group colors = %d/%d, want same", a.GroupColorIndex, b.GroupColorIndex)
	}
}

func TestAnnotateFrameUsesCustomFunctionDefinition(t *testing.T) {
	raw, frame := mustEncodedFrame(t, Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 2,
		Priority:      2,
		SourceAddress: 1,
		MessageNumber: 1,
		GroupNumber:   0,
		Data:          []byte{0x08, 0x34, 0x12, 0x7f},
	})
	custom := []CustomFunctionDefinition{{
		Code:      0x08,
		Name:      "厂商测试",
		Direction: "controller_to_device",
		Fields: []CustomDataFieldDefinition{
			{Key: "value", Label: "测试值", Offset: 1, Length: 2, Type: "uint16", Endian: "little"},
			{Key: "flag", Label: "标志", Offset: 3, Length: 1, Type: "uint8", Enum: map[byte]string{0x7f: "开启"}},
		},
	}}

	annotated := AnnotateFrame(FrameRecord{SessionID: "s1", Direction: "tx", Frame: frame, Hex: FormatHex(raw)}, custom, nil)
	if annotated.Function.Name != "厂商测试" {
		t.Fatalf("function name = %q, want custom name", annotated.Function.Name)
	}
	assertField(t, annotated.DataFields, "value", 9, 11, "4660")
	assertField(t, annotated.DataFields, "flag", 11, 12, "开启")
}

func TestAnnotateQueryAnswerFieldsFromAppendixC(t *testing.T) {
	raw, frame := mustEncodedFrame(t, Frame{
		Type:          FrameTypeAnswer,
		TargetAddress: 1,
		Priority:      3,
		SourceAddress: 2,
		MessageNumber: 3,
		GroupNumber:   0,
		Data:          []byte{byte(FunctionQueryProtocolVersion), 0x01, 0x02},
	})

	annotated := AnnotateFrame(FrameRecord{SessionID: "s1", Direction: "rx", Frame: frame, Hex: FormatHex(raw)}, nil, nil)
	assertField(t, annotated.DataFields, "version_major", 9, 10, "1")
	assertField(t, annotated.DataFields, "version_minor", 10, 11, "2")
}

func TestAnnotateEventAnswerUsesAppendixCEnums(t *testing.T) {
	raw, frame := mustEncodedFrame(t, Frame{
		Type:          FrameTypeAnswer,
		TargetAddress: 1,
		Priority:      3,
		SourceAddress: 2,
		MessageNumber: 4,
		GroupNumber:   1,
		Data:          []byte{byte(FunctionQueryCurrentEvent), 0x01, 0x02, 0x03, 0x04, 0xa3, 0x00, 0x03},
	})

	annotated := AnnotateFrame(FrameRecord{SessionID: "s1", Direction: "rx", Frame: frame, Hex: FormatHex(raw)}, nil, nil)
	assertField(t, annotated.DataFields, "device_type", 13, 15, "消防电气控制装置")
}

func TestAnnotateParameterAnswerUsesAppendixCEnums(t *testing.T) {
	raw, frame := mustEncodedFrame(t, Frame{
		Type:          FrameTypeAnswer,
		TargetAddress: 1,
		Priority:      3,
		SourceAddress: 2,
		MessageNumber: 5,
		GroupNumber:   0,
		Data:          []byte{byte(FunctionQueryParameter), 0x01, 0x34, 0x12},
	})

	annotated := AnnotateFrame(FrameRecord{SessionID: "s1", Direction: "rx", Frame: frame, Hex: FormatHex(raw)}, nil, nil)
	assertField(t, annotated.DataFields, "parameter_type", 9, 10, "高度")
	assertField(t, annotated.DataFields, "parameter_value", 10, 12, "4660")
}

func TestBuildDataFromFieldsUsesAppendixCAndCustomDefinitions(t *testing.T) {
	data, err := BuildDataFromFields(FunctionQueryDeviceStatus, map[string]any{
		"controller_id": 3,
		"unit_id":       4,
	}, nil)
	if err != nil {
		t.Fatalf("BuildDataFromFields() error = %v", err)
	}
	assertHex(t, data, "220304")

	custom := []CustomFunctionDefinition{{
		Code: 0x08,
		Name: "厂商测试",
		Fields: []CustomDataFieldDefinition{
			{Key: "value", Label: "测试值", Offset: 1, Length: 2, Type: "uint16", Endian: "little"},
			{Key: "flag", Label: "标志", Offset: 3, Length: 1, Type: "uint8"},
		},
	}}
	data, err = BuildDataFromFields(0x08, map[string]any{"value": 0x1234, "flag": 0x7f}, custom)
	if err != nil {
		t.Fatalf("BuildDataFromFields(custom) error = %v", err)
	}
	assertHex(t, data, "0834127f")
}

func assertSegment(t *testing.T, segments []FrameSegment, key string, start int, end int, labelPart string) {
	t.Helper()
	for _, segment := range segments {
		if segment.Key == key {
			if segment.Start != start || segment.End != end {
				t.Fatalf("segment %s range = %d..%d, want %d..%d", key, segment.Start, segment.End, start, end)
			}
			if !strings.Contains(segment.Label, labelPart) && !strings.Contains(segment.Meaning, labelPart) {
				t.Fatalf("segment %s label/meaning = %q/%q, want %q", key, segment.Label, segment.Meaning, labelPart)
			}
			return
		}
	}
	t.Fatalf("segment %s not found in %#v", key, segments)
}

func assertField(t *testing.T, fields []FieldAnnotation, key string, start int, end int, meaningPart string) {
	t.Helper()
	for _, field := range fields {
		if field.Key == key {
			if field.Start != start || field.End != end {
				t.Fatalf("field %s range = %d..%d, want %d..%d", key, field.Start, field.End, start, end)
			}
			if !strings.Contains(field.Label, meaningPart) && !strings.Contains(field.Meaning, meaningPart) && !strings.Contains(field.ValueText, meaningPart) {
				t.Fatalf("field %s label/meaning/value = %q/%q/%q, want %q", key, field.Label, field.Meaning, field.ValueText, meaningPart)
			}
			return
		}
	}
	t.Fatalf("field %s not found in %#v", key, fields)
}

func mustEncodedFrame(t *testing.T, frame Frame) ([]byte, Frame) {
	t.Helper()
	raw, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	decoded, err := DecodeFrame(raw)
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	return raw, decoded
}
