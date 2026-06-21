package serial

import (
	"fmt"
	"strings"
	"testing"
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
