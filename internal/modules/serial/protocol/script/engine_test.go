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
