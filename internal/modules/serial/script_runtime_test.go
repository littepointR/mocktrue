package serial

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestSerialScriptRuntimeFunctionAPI(t *testing.T) {
	runtime := newSerialScriptRuntime(map[string]any{
		"script": `
			var bytes = input.bytes();
			field("hex", input.hex());
			field("text", input.text("utf-8"));
			field("sum", sum8(bytes));
			field("crc", crc16(bytes));
			field("now", now() > 0);
			state.set("runs", (state.get("runs") || 0) + 1);
			field("runs", state.get("runs"));
			state.delete("missing");
			output.bytes([bytes[1], bytes[0]]);
		`,
		"timeoutMs":      100,
		"maxOutputBytes": 8,
		"maxStateBytes":  128,
		"encoding":       "utf-8",
	})

	first, err := runtime.run(serialScriptRunInput{Data: []byte("AB")})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if string(first.Output) != "BA" {
		t.Fatalf("output = %q, want BA", string(first.Output))
	}
	assertScriptField(t, first.Fields, "hex", "41 42")
	assertScriptField(t, first.Fields, "text", "AB")
	assertScriptField(t, first.Fields, "sum", "131")
	assertScriptField(t, first.Fields, "now", "true")
	assertScriptField(t, first.Fields, "runs", "1")

	second, err := runtime.run(serialScriptRunInput{Data: []byte("CD")})
	if err != nil {
		t.Fatalf("second run returned error: %v", err)
	}
	assertScriptField(t, second.Fields, "runs", "2")
}

func TestSerialScriptRuntimeDocumentationExamples(t *testing.T) {
	t.Run("line generator", func(t *testing.T) {
		runtime := newSerialScriptRuntimeForNode("serial.script.generator", map[string]any{
			"script":    `output.text("tick\n", "utf-8");`,
			"timeoutMs": 100,
		})
		result, err := runtime.run(serialScriptRunInput{})
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
		if string(result.Output) != "tick\n" {
			t.Fatalf("generator output = %q, want tick newline", string(result.Output))
		}
	})

	t.Run("pass through transform", func(t *testing.T) {
		runtime := newSerialScriptRuntimeForNode("serial.script.transform", map[string]any{
			"script":    `output.bytes(input.bytes());`,
			"timeoutMs": 100,
		})
		input := []byte{0xde, 0xad, 0x01, 0x02}
		result, err := runtime.run(serialScriptRunInput{Data: input})
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
		if string(result.Output) != string(input) {
			t.Fatalf("transform output = % x, want % x", result.Output, input)
		}
	})

	t.Run("text analyzer with JavaScript string APIs", func(t *testing.T) {
		runtime := newSerialScriptRuntimeForNode("serial.script.analyzer", map[string]any{
			"script": `
				var text = input.text("utf-8");
				var match = text.match(/^TEMP=(-?\d+(?:\.\d+)?)$/);
				if (match) {
					field("temperature_c", Number(match[1]), match[1] + " °C");
				} else {
					error("expected TEMP=<number>");
				}
			`,
			"timeoutMs": 100,
		})
		result, err := runtime.run(serialScriptRunInput{Data: []byte("TEMP=21.5")})
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
		if len(result.Errors) != 0 {
			t.Fatalf("errors = %#v, want none", result.Errors)
		}
		assertScriptField(t, result.Fields, "temperature_c", "21.5")
	})

	t.Run("binary length and checksum analyzer", func(t *testing.T) {
		runtime := newSerialScriptRuntimeForNode("serial.script.analyzer", map[string]any{
			"script": `
				var bytes = input.bytes();
				field("length", bytes.length);
				if (bytes.length < 2) {
					error("frame too short");
				} else {
					var payload = [];
					for (var i = 0; i < bytes.length - 1; i++) {
						payload.push(bytes[i]);
					}
					var actual = sum8(payload);
					var expected = bytes[bytes.length - 1];
					field("checksum", actual, actual === expected ? "OK" : "mismatch");
					if (actual !== expected) {
						error("checksum mismatch");
					}
				}
			`,
			"timeoutMs": 100,
		})
		result, err := runtime.run(serialScriptRunInput{Data: []byte{0x01, 0x02, 0x03}})
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
		if len(result.Errors) != 0 {
			t.Fatalf("errors = %#v, want none", result.Errors)
		}
		assertScriptField(t, result.Fields, "length", "3")
		assertScriptField(t, result.Fields, "checksum", "3")

		result, err = runtime.run(serialScriptRunInput{Data: []byte{0x01, 0x02, 0x00}})
		if err != nil {
			t.Fatalf("bad checksum run returned Go error: %v", err)
		}
		if len(result.Errors) != 1 || result.Errors[0] != "checksum mismatch" {
			t.Fatalf("bad checksum errors = %#v, want checksum mismatch", result.Errors)
		}
	})
}

func TestSerialScriptRuntimeEnforcesOutputAndStateLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config map[string]any
		want   string
	}{
		{
			name: "output",
			config: map[string]any{
				"script":         `output.bytes([1, 2, 3]);`,
				"timeoutMs":      100,
				"maxOutputBytes": 2,
			},
			want: "output exceeds",
		},
		{
			name: "state",
			config: map[string]any{
				"script":        `state.set("large", "1234567890");`,
				"timeoutMs":     100,
				"maxStateBytes": 8,
			},
			want: "state exceeds",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := newSerialScriptRuntime(tc.config)
			_, err := runtime.run(serialScriptRunInput{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("run error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSerialScriptStateGetReturnsClone(t *testing.T) {
	state := newSerialScriptState(64)
	if err := state.set("payload", map[string]any{"items": []any{"ok"}}); err != nil {
		t.Fatalf("set returned error: %v", err)
	}

	got, ok := state.get("payload").(map[string]any)
	if !ok {
		t.Fatalf("state payload type = %T, want map[string]any", state.get("payload"))
	}
	got["large"] = strings.Repeat("x", 128)

	stored, ok := state.get("payload").(map[string]any)
	if !ok {
		t.Fatalf("stored payload type = %T, want map[string]any", state.get("payload"))
	}
	if _, exists := stored["large"]; exists {
		t.Fatalf("state.get returned mutable stored object reference")
	}
}

func TestSerialScriptStateSetStoresClone(t *testing.T) {
	state := newSerialScriptState(64)
	payload := map[string]any{"items": []any{"ok"}}
	if err := state.set("payload", payload); err != nil {
		t.Fatalf("set returned error: %v", err)
	}
	payload["large"] = strings.Repeat("x", 128)

	stored, ok := state.get("payload").(map[string]any)
	if !ok {
		t.Fatalf("stored payload type = %T, want map[string]any", state.get("payload"))
	}
	if _, exists := stored["large"]; exists {
		t.Fatalf("state.set stored mutable caller object reference")
	}
}

func TestSerialScriptRuntimeExposesAPIsByNodeType(t *testing.T) {
	for _, tc := range []struct {
		name     string
		nodeType string
		script   string
		input    []byte
		wantOut  string
	}{
		{
			name:     "generator has output only",
			nodeType: "serial.script.generator",
			script:   `field("hasInput", typeof input); output.text("ok");`,
			wantOut:  "ok",
		},
		{
			name:     "analyzer has input only",
			nodeType: "serial.script.analyzer",
			script:   `field("hasOutput", typeof output); field("hex", input.hex());`,
			input:    []byte{0x01, 0x02},
		},
		{
			name:     "transform has input and output",
			nodeType: "serial.script.transform",
			script:   `output.hex(input.hex());`,
			input:    []byte{0x03, 0x04},
			wantOut:  string([]byte{0x03, 0x04}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := newSerialScriptRuntimeForNode(tc.nodeType, map[string]any{
				"script":    tc.script,
				"timeoutMs": 100,
				"encoding":  "utf-8",
			})

			result, err := runtime.run(serialScriptRunInput{Data: tc.input})
			if err != nil {
				t.Fatalf("run returned error: %v", err)
			}
			if string(result.Output) != tc.wantOut {
				t.Fatalf("output = %q, want %q", string(result.Output), tc.wantOut)
			}
			switch tc.nodeType {
			case "serial.script.generator":
				assertScriptField(t, result.Fields, "hasInput", "undefined")
			case "serial.script.analyzer":
				assertScriptField(t, result.Fields, "hasOutput", "undefined")
				assertScriptField(t, result.Fields, "hex", "01 02")
			}
		})
	}
}

func TestSerialScriptRuntimeRejectsUnavailableAPIsByNodeType(t *testing.T) {
	for _, tc := range []struct {
		name     string
		nodeType string
		script   string
		input    []byte
		want     string
	}{
		{
			name:     "generator cannot read input bytes",
			nodeType: "serial.script.generator",
			script:   `input.bytes();`,
			want:     "input is not defined",
		},
		{
			name:     "generator cannot read input hex",
			nodeType: "serial.script.generator",
			script:   `input.hex();`,
			want:     "input is not defined",
		},
		{
			name:     "generator cannot read input text",
			nodeType: "serial.script.generator",
			script:   `input.text("utf-8");`,
			want:     "input is not defined",
		},
		{
			name:     "analyzer cannot write output bytes",
			nodeType: "serial.script.analyzer",
			script:   `output.bytes([1]);`,
			input:    []byte{0x01},
			want:     "output is not defined",
		},
		{
			name:     "analyzer cannot write output hex",
			nodeType: "serial.script.analyzer",
			script:   `output.hex("01");`,
			input:    []byte{0x01},
			want:     "output is not defined",
		},
		{
			name:     "analyzer cannot write output text",
			nodeType: "serial.script.analyzer",
			script:   `output.text("x");`,
			input:    []byte{0x01},
			want:     "output is not defined",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := newSerialScriptRuntimeForNode(tc.nodeType, map[string]any{
				"script":    tc.script,
				"timeoutMs": 100,
				"encoding":  "utf-8",
			})

			_, err := runtime.run(serialScriptRunInput{Data: tc.input})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("run error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSerialScriptRuntimeUsesDefaultsForNonPositiveLimits(t *testing.T) {
	runtime := newSerialScriptRuntimeForNode("serial.script.transform", map[string]any{
		"script":         `output.text("ok");`,
		"timeoutMs":      0,
		"maxOutputBytes": 0,
		"maxStateBytes":  0,
	})

	if runtime.timeout != serialScriptDefaultTimeout {
		t.Fatalf("timeout = %s, want default %s", runtime.timeout, serialScriptDefaultTimeout)
	}
	if runtime.maxOutputSize != serialScriptDefaultMaxOutputSize {
		t.Fatalf("maxOutputSize = %d, want default %d", runtime.maxOutputSize, serialScriptDefaultMaxOutputSize)
	}
	if runtime.state.maxSize != serialScriptDefaultMaxStateSize {
		t.Fatalf("state maxSize = %d, want default %d", runtime.state.maxSize, serialScriptDefaultMaxStateSize)
	}
}

func TestSerialScriptRuntimeCoversErrorBranchesAndDisplayFields(t *testing.T) {
	t.Run("empty script", func(t *testing.T) {
		runtime := newSerialScriptRuntime(map[string]any{"script": "   "})
		if _, err := runtime.run(serialScriptRunInput{}); err == nil || !strings.Contains(err.Error(), "script must not be empty") {
			t.Fatalf("empty script error = %v, want validation error", err)
		}
	})

	for _, tc := range []struct {
		name   string
		script string
		input  []byte
		limit  int
		want   string
	}{
		{name: "input text unsupported encoding", script: `input.text("unsupported");`, input: []byte("abc"), want: "unsupported text encoding"},
		{name: "output hex invalid", script: `output.hex("zz");`, want: "invalid byte"},
		{name: "output text unsupported ascii rune", script: `output.text("é", "ascii");`, want: "not supported by ascii"},
		{name: "output text over limit", script: `output.text("too long");`, limit: 2, want: "output exceeds"},
		{name: "output bytes invalid numeric", script: `output.bytes([256]);`, want: "not numeric"},
		{name: "crc16 invalid bytes", script: `crc16(["bad"]);`, want: "not numeric"},
		{name: "sum8 invalid bytes", script: `sum8(["bad"]);`, want: "not numeric"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			maxOutputBytes := tc.limit
			if maxOutputBytes == 0 {
				maxOutputBytes = 64
			}
			runtime := newSerialScriptRuntime(map[string]any{
				"script":         tc.script,
				"timeoutMs":      100,
				"maxOutputBytes": maxOutputBytes,
			})
			_, err := runtime.run(serialScriptRunInput{Data: tc.input})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("run error = %v, want %q", err, tc.want)
			}
		})
	}

	t.Run("field display and empty outputs", func(t *testing.T) {
		runtime := newSerialScriptRuntime(map[string]any{
			"script":    `field("speed", 115200, "115.2 kbps"); output.bytes(null); output.bytes([]); drop();`,
			"timeoutMs": 100,
		})
		result, err := runtime.run(serialScriptRunInput{})
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
		if len(result.Output) != 0 || !result.Drop {
			t.Fatalf("result output=% x drop=%v, want dropped empty output", result.Output, result.Drop)
		}
		if len(result.Fields) != 1 || result.Fields[0].Name != "speed" || result.Fields[0].Display != "115.2 kbps" {
			t.Fatalf("fields = %#v, want speed field with display text", result.Fields)
		}
	})
}

func TestSerialScriptStateRejectsUncloneableValues(t *testing.T) {
	state := newSerialScriptState(64)
	state.values["bad"] = func() {}
	if got := state.get("bad"); got != nil {
		t.Fatalf("state.get uncloneable value = %#v, want nil", got)
	}
	if err := state.set("bad", func() {}); err == nil {
		t.Fatalf("state.set must reject values that cannot be JSON cloned")
	}
	if _, err := jsonEncodedSize(func() {}); err == nil {
		t.Fatalf("jsonEncodedSize must reject values that cannot be JSON encoded")
	}
	if _, err := jsonCompatibleClone(func() {}); err == nil {
		t.Fatalf("jsonCompatibleClone must reject values that cannot be JSON cloned")
	}
}

func TestGojaValueToBytesCoversSupportedAndInvalidValues(t *testing.T) {
	vm := goja.New()
	tests := []struct {
		name    string
		value   goja.Value
		want    []byte
		wantErr string
	}{
		{name: "undefined", value: goja.Undefined()},
		{name: "null", value: goja.Null()},
		{name: "bytes", value: vm.ToValue([]byte{0x01, 0x02}), want: []byte{0x01, 0x02}},
		{name: "ints", value: vm.ToValue([]int{0, 255}), want: []byte{0x00, 0xff}},
		{name: "int out of range", value: vm.ToValue([]int{256}), wantErr: "out of range"},
		{name: "any numeric", value: vm.ToValue([]any{float64(1), int64(2), uint8(3)}), want: []byte{0x01, 0x02, 0x03}},
		{name: "any nonnumeric", value: vm.ToValue([]any{"x"}), wantErr: "not numeric"},
		{name: "string", value: vm.ToValue("abc"), want: []byte("abc")},
		{name: "unsupported", value: vm.ToValue(true), wantErr: "unsupported byte value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := gojaValueToBytes(tt.value)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("gojaValueToBytes error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("gojaValueToBytes returned error: %v", err)
			}
			if string(got) != string(tt.want) {
				t.Fatalf("gojaValueToBytes = % x, want % x", got, tt.want)
			}
		})
	}
}

func TestNumericByteCoversIntegerAndFloatShapes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int
		ok    bool
	}{
		{name: "int", value: int(255), want: 255, ok: true},
		{name: "int negative", value: int(-1)},
		{name: "int8", value: int8(127), want: 127, ok: true},
		{name: "int8 negative", value: int8(-1)},
		{name: "int16", value: int16(255), want: 255, ok: true},
		{name: "int16 too large", value: int16(256)},
		{name: "int32", value: int32(255), want: 255, ok: true},
		{name: "int32 too large", value: int32(256)},
		{name: "int64", value: int64(255), want: 255, ok: true},
		{name: "int64 too large", value: int64(256)},
		{name: "uint", value: uint(255), want: 255, ok: true},
		{name: "uint too large", value: uint(256)},
		{name: "uint8", value: uint8(7), want: 7, ok: true},
		{name: "uint16", value: uint16(255), want: 255, ok: true},
		{name: "uint16 too large", value: uint16(256)},
		{name: "uint32", value: uint32(255), want: 255, ok: true},
		{name: "uint32 too large", value: uint32(256)},
		{name: "uint64", value: uint64(255), want: 255, ok: true},
		{name: "uint64 too large", value: uint64(256)},
		{name: "float32 integer", value: float32(42), want: 42, ok: true},
		{name: "float32 fractional", value: float32(42.5)},
		{name: "float64 integer", value: float64(43), want: 43, ok: true},
		{name: "float64 fractional", value: float64(43.5)},
		{name: "unsupported", value: "44"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := numericByte(tt.value)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Fatalf("numericByte(%T(%v)) = %d, %v; want %d, %v", tt.value, tt.value, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func assertScriptField(t *testing.T, fields []serialScriptField, name string, want string) {
	t.Helper()
	for _, field := range fields {
		if field.Name == name {
			if got := fmt.Sprint(field.Value); got != want {
				t.Fatalf("field %s = %q, want %q", name, got, want)
			}
			return
		}
	}
	t.Fatalf("field %s not found in %#v", name, fields)
}
