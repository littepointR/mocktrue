package modbus

import "testing"

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
