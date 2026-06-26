package script

import (
	"testing"
	"time"
)

func TestEngineRunSimpleScript(t *testing.T) {
	t.Parallel()
	engine := NewEngine()

	script := `
		field("byte0", byte(0));
		field("byte1", byte(1));
	`
	data := []byte{0x01, 0x02}

	result, err := engine.RunScript(script, data, time.Second)
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
	if !result.OK {
		t.Fatalf("not OK: %v", result.Errors)
	}
	if len(result.Fields) != 2 {
		t.Fatalf("Fields len = %d, want 2", len(result.Fields))
	}
	if result.Fields[0].Name != "byte0" || result.Fields[0].Value != int64(1) {
		t.Fatalf("Field[0] = %+v", result.Fields[0])
	}
}

func TestEngineTimeout(t *testing.T) {
	t.Parallel()
	engine := NewEngine()

	script := `while(true) {}`

	_, err := engine.RunScript(script, []byte{0x01}, 100*time.Millisecond)
	if err == nil {
		t.Fatalf("should have timed out")
	}
}

func TestEngineFieldWithDisplay(t *testing.T) {
	t.Parallel()
	engine := NewEngine()

	script := `
		var v = byte(0);
		field("val", v, "0x" + v.toString(16));
	`
	data := []byte{0x0A}

	result, err := engine.RunScript(script, data, time.Second)
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
	if result.Fields[0].Display != "0xa" {
		t.Fatalf("Display = %q, want 0xa", result.Fields[0].Display)
	}
}

func TestEngineByteArrayAndIntegerHelpers(t *testing.T) {
	t.Parallel()
	engine := NewEngine()

	script := `
		field("bytes", bytes(-1, 4).join(","));
		field("u16le", u16(1, true));
		field("u16be", u16(1, false));
		field("u32le", u32(0, true));
		field("u32be", u32(0, false));
		field("u32short", u32(2, true));
		field("len", len());
	`

	result, err := engine.RunScript(script, []byte{0x01, 0x02, 0x03, 0x04}, time.Second)
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
	if !result.OK {
		t.Fatalf("RunScript not OK: %v", result.Errors)
	}

	got := map[string]any{}
	for _, field := range result.Fields {
		got[field.Name] = field.Value
	}
	checks := map[string]any{
		"bytes":    "0,1,2,3",
		"u16le":    int64(0x0302),
		"u16be":    int64(0x0203),
		"u32le":    int64(0x04030201),
		"u32be":    int64(0x01020304),
		"u32short": int64(0),
		"len":      int64(4),
	}
	for name, want := range checks {
		if got[name] != want {
			t.Fatalf("field %s = %#v, want %#v", name, got[name], want)
		}
	}
}

func TestEngineRecordsScriptErrors(t *testing.T) {
	t.Parallel()
	engine := NewEngine()

	result, err := engine.RunScript(`error("bad frame")`, []byte{0x01}, time.Second)
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
	if result.OK {
		t.Fatalf("RunScript OK = true, want false")
	}
	if len(result.Errors) != 1 || result.Errors[0] != "bad frame" {
		t.Fatalf("Errors = %#v, want bad frame", result.Errors)
	}
}
