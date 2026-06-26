package modbus

import (
	"testing"
	"time"
)

func TestRegisterValueEncodingAndDecoding(t *testing.T) {
	tests := []struct {
		name        string
		spec        RegisterValueSpec
		input       string
		wantRegs    []uint16
		wantDisplay string
	}{
		{
			name:        "uint16",
			spec:        RegisterValueSpec{DataType: DataTypeUint16},
			input:       "42",
			wantRegs:    []uint16{42},
			wantDisplay: "42",
		},
		{
			name:        "int16",
			spec:        RegisterValueSpec{DataType: DataTypeInt16},
			input:       "-2",
			wantRegs:    []uint16{0xfffe},
			wantDisplay: "-2",
		},
		{
			name:        "int32 big word order",
			spec:        RegisterValueSpec{DataType: DataTypeInt32, WordOrder: WordOrderBig},
			input:       "305419896",
			wantRegs:    []uint16{0x1234, 0x5678},
			wantDisplay: "305419896",
		},
		{
			name:        "int32 little word order",
			spec:        RegisterValueSpec{DataType: DataTypeInt32, WordOrder: WordOrderLittle},
			input:       "305419896",
			wantRegs:    []uint16{0x5678, 0x1234},
			wantDisplay: "305419896",
		},
		{
			name:        "float",
			spec:        RegisterValueSpec{DataType: DataTypeFloat, WordOrder: WordOrderBig},
			input:       "1.5",
			wantRegs:    []uint16{0x3fc0, 0x0000},
			wantDisplay: "1.5",
		},
		{
			name:        "double",
			spec:        RegisterValueSpec{DataType: DataTypeDouble, WordOrder: WordOrderBig},
			input:       "1.25",
			wantRegs:    []uint16{0x3ff4, 0x0000, 0x0000, 0x0000},
			wantDisplay: "1.25",
		},
		{
			name:        "unix",
			spec:        RegisterValueSpec{DataType: DataTypeUnix, WordOrder: WordOrderBig},
			input:       "1700000000",
			wantRegs:    []uint16{0x6553, 0xf100},
			wantDisplay: "2023/11/14 22:13:20",
		},
		{
			name:        "datetime",
			spec:        RegisterValueSpec{DataType: DataTypeDateTime, WordOrder: WordOrderBig},
			input:       "1700000000000",
			wantRegs:    []uint16{0x0017, 0x0b0e, 0x160d, 0x4e20},
			wantDisplay: "2023/11/14 22:13:20",
		},
		{
			name:        "utf8",
			spec:        RegisterValueSpec{DataType: DataTypeUTF8, Length: 3},
			input:       "Hi",
			wantRegs:    []uint16{0x4869, 0x0000, 0x0000},
			wantDisplay: "Hi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regs, err := EncodeRegisterValue(tt.spec, tt.input)
			if err != nil {
				t.Fatalf("EncodeRegisterValue() error = %v", err)
			}
			if !equalU16(regs, tt.wantRegs) {
				t.Fatalf("registers = %#v, want %#v", regs, tt.wantRegs)
			}
			decoded, err := DecodeRegisterValue(tt.spec, regs)
			if err != nil {
				t.Fatalf("DecodeRegisterValue() error = %v", err)
			}
			if decoded.Display != tt.wantDisplay {
				t.Fatalf("display = %q, want %q", decoded.Display, tt.wantDisplay)
			}
		})
	}
}

func TestDecodeRegisterValueAppliesScalingAndInterpolation(t *testing.T) {
	decoded, err := DecodeRegisterValue(RegisterValueSpec{
		DataType:      DataTypeUint16,
		ScalingFactor: 0.1,
		Interpolate: &LinearInterpolation{
			X1: 0,
			X2: 10,
			Y1: 100,
			Y2: 200,
		},
	}, []uint16{50})
	if err != nil {
		t.Fatalf("DecodeRegisterValue() error = %v", err)
	}
	if decoded.Display != "150" {
		t.Fatalf("display = %q, want 150", decoded.Display)
	}
	if decoded.Numeric != 150 {
		t.Fatalf("numeric = %v, want 150", decoded.Numeric)
	}
}

func TestGroupRegisterMappings(t *testing.T) {
	groups := GroupRegisterMappings([]RegisterMapping{
		{Address: 0, DataType: DataTypeUint16},
		{Address: 2, DataType: DataTypeInt32},
		{Address: 8, DataType: DataTypeUTF8, Length: 4, GroupEnd: true},
		{Address: 20, DataType: DataTypeFloat},
	}, 12)

	want := []AddressGroup{
		{Address: 0, Quantity: 12},
		{Address: 20, Quantity: 2},
	}
	if len(groups) != len(want) {
		t.Fatalf("groups = %#v, want %#v", groups, want)
	}
	for i := range groups {
		if groups[i] != want[i] {
			t.Fatalf("groups[%d] = %#v, want %#v", i, groups[i], want[i])
		}
	}
}

func TestRegisterValueConversionEdges(t *testing.T) {
	tests := []struct {
		name        string
		spec        RegisterValueSpec
		input       string
		wantRegs    []uint16
		wantDisplay string
	}{
		{
			name:        "uint32",
			spec:        RegisterValueSpec{DataType: DataTypeUint32},
			input:       "4294967295",
			wantRegs:    []uint16{0xffff, 0xffff},
			wantDisplay: "4294967295",
		},
		{
			name:        "int64",
			spec:        RegisterValueSpec{DataType: DataTypeInt64},
			input:       "-2",
			wantRegs:    []uint16{0xffff, 0xffff, 0xffff, 0xfffe},
			wantDisplay: "-2",
		},
		{
			name:        "uint64",
			spec:        RegisterValueSpec{DataType: DataTypeUint64},
			input:       "72623859790382856",
			wantRegs:    []uint16{0x0102, 0x0304, 0x0506, 0x0708},
			wantDisplay: "72623859790382848",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regs, err := EncodeRegisterValue(tt.spec, tt.input)
			if err != nil {
				t.Fatalf("EncodeRegisterValue() error = %v", err)
			}
			if !equalU16(regs, tt.wantRegs) {
				t.Fatalf("registers = %#v, want %#v", regs, tt.wantRegs)
			}
			decoded, err := DecodeRegisterValue(tt.spec, regs)
			if err != nil {
				t.Fatalf("DecodeRegisterValue() error = %v", err)
			}
			if decoded.Display != tt.wantDisplay {
				t.Fatalf("display = %q, want %q", decoded.Display, tt.wantDisplay)
			}
		})
	}

	decoded, err := DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeUTF8, Length: 1}, []uint16{0xfffe})
	if err != nil {
		t.Fatalf("DecodeRegisterValue(invalid UTF-8) error = %v", err)
	}
	if decoded.Display != "" {
		t.Fatalf("invalid UTF-8 display = %q, want empty", decoded.Display)
	}

	decoded, err = DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeFloat}, []uint16{0x7fc0, 0x0000})
	if err != nil {
		t.Fatalf("DecodeRegisterValue(NaN) error = %v", err)
	}
	if decoded.Display != "NaN" {
		t.Fatalf("NaN display = %q, want NaN", decoded.Display)
	}

	decoded, err = DecodeRegisterValue(RegisterValueSpec{
		DataType:    DataTypeUint16,
		Interpolate: &LinearInterpolation{X1: 10, X2: 10, Y1: 7, Y2: 99},
	}, []uint16{123})
	if err != nil {
		t.Fatalf("DecodeRegisterValue(zero-span interpolation) error = %v", err)
	}
	if decoded.Display != "7" || decoded.Numeric != 7 {
		t.Fatalf("zero-span interpolation decoded = %#v, want 7", decoded)
	}

	originalLocal := time.Local
	localZone := time.FixedZone("mocktrue-localtime-test", 90*60)
	time.Local = localZone
	t.Cleanup(func() { time.Local = originalLocal })

	wantLocalDisplay := time.Unix(1, 0).In(localZone).Format("2006/01/02 15:04:05")
	utcDisplay := time.Unix(1, 0).UTC().Format("2006/01/02 15:04:05")
	if wantLocalDisplay == utcDisplay {
		t.Fatalf("test local display = UTC display %q; fixed local zone must differ from UTC", wantLocalDisplay)
	}

	decoded, err = DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeUnix, LocalTime: true}, []uint16{0, 1})
	if err != nil {
		t.Fatalf("DecodeRegisterValue(local unix) error = %v", err)
	}
	if decoded.Display != wantLocalDisplay {
		t.Fatalf("local unix display = %q, want %q", decoded.Display, wantLocalDisplay)
	}
}

func TestRegisterValueConversionErrorsAndDatetimeBounds(t *testing.T) {
	if _, err := EncodeRegisterValue(RegisterValueSpec{DataType: DataTypeNone}, "1"); err == nil {
		t.Fatalf("EncodeRegisterValue(none) error = nil, want error")
	}
	if _, err := EncodeRegisterValue(RegisterValueSpec{DataType: DataTypeUint16}, "not-a-number"); err == nil {
		t.Fatalf("EncodeRegisterValue(invalid number) error = nil, want error")
	}
	if _, err := DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeNone}, []uint16{1}); err == nil {
		t.Fatalf("DecodeRegisterValue(none) error = nil, want error")
	}
	if _, err := DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeDouble}, []uint16{1, 2}); err == nil {
		t.Fatalf("DecodeRegisterValue(short double) error = nil, want error")
	}

	encoded, err := EncodeRegisterValue(RegisterValueSpec{DataType: DataTypeDateTime}, "1")
	if err != nil {
		t.Fatalf("EncodeRegisterValue(datetime before lower bound) error = %v", err)
	}
	decoded, err := DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeDateTime}, encoded)
	if err != nil {
		t.Fatalf("DecodeRegisterValue(datetime before lower bound) error = %v", err)
	}
	if decoded.Display != "2000/01/01 00:00:00" {
		t.Fatalf("bounded datetime display = %q, want 2000/01/01 00:00:00", decoded.Display)
	}

	decoded, err = DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeDateTime}, []uint16{0xffff, 0xffff, 0xffff, 0xffff})
	if err != nil {
		t.Fatalf("DecodeRegisterValue(blank datetime) error = %v", err)
	}
	if decoded.Display != "" {
		t.Fatalf("blank datetime display = %q, want empty", decoded.Display)
	}

	decoded, err = DecodeRegisterValue(RegisterValueSpec{DataType: DataTypeDateTime}, []uint16{0x0017, 0x0d01, 0x0000, 0x0000})
	if err != nil {
		t.Fatalf("DecodeRegisterValue(invalid datetime fields) error = %v", err)
	}
	if decoded.Display != "" {
		t.Fatalf("invalid datetime display = %q, want empty", decoded.Display)
	}
}

func TestGroupRegisterMappingsSkipsNoneAndSplitsAtLimits(t *testing.T) {
	groups := GroupRegisterMappings([]RegisterMapping{
		{Address: 0, DataType: DataTypeNone},
		{Address: 1, DataType: DataTypeUint16},
		{Address: 2, DataType: DataTypeNone},
		{Address: 5, DataType: DataTypeDouble},
		{Address: 15, DataType: DataTypeFloat},
	}, 4)
	want := []AddressGroup{
		{Address: 1, Quantity: 1},
		{Address: 5, Quantity: 4},
		{Address: 15, Quantity: 2},
	}
	if len(groups) != len(want) {
		t.Fatalf("groups = %#v, want %#v", groups, want)
	}
	for i := range groups {
		if groups[i] != want[i] {
			t.Fatalf("groups[%d] = %#v, want %#v", i, groups[i], want[i])
		}
	}
	if end := mappingEnd(RegisterMapping{Address: 42, DataType: DataTypeNone}); end != 42 {
		t.Fatalf("mappingEnd(none) = %d, want 42", end)
	}
	if groups := GroupRegisterMappings([]RegisterMapping{{Address: 3, DataType: DataTypeUint16}}, 0); len(groups) != 1 || groups[0] != (AddressGroup{Address: 3, Quantity: 1}) {
		t.Fatalf("GroupRegisterMappings(default max) = %#v", groups)
	}
}
