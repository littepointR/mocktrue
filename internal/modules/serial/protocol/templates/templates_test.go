package templates

import (
	"testing"

	"github.com/littepointR/mocktrue/internal/modules/serial/protocol"
)

func TestGetTemplateModbus(t *testing.T) {
	t.Parallel()
	tmpl := GetTemplate("Modbus RTU")
	if tmpl == nil {
		t.Fatalf("Modbus RTU template not found")
	}
	if tmpl.Kind != protocol.KindVisual {
		t.Fatalf("Kind = %q, want visual", tmpl.Kind)
	}
	if tmpl.Config.Visual == nil {
		t.Fatalf("Visual config is nil")
	}
}

func TestGetTemplateAA55(t *testing.T) {
	t.Parallel()
	tmpl := GetTemplate("AA55 自定义帧")
	if tmpl == nil {
		t.Fatalf("AA55 template not found")
	}
	if tmpl.Config.Visual == nil || len(tmpl.Config.Visual.Header) != 2 {
		t.Fatalf("AA55 header missing")
	}
}

func TestGetTemplateNMEA(t *testing.T) {
	t.Parallel()
	tmpl := GetTemplate("NMEA")
	if tmpl == nil {
		t.Fatalf("NMEA template not found")
	}
	if tmpl.Kind != protocol.KindScript {
		t.Fatalf("Kind = %q, want script", tmpl.Kind)
	}
	if tmpl.Config.Script == "" {
		t.Fatalf("Script is empty")
	}
}

func TestGetTemplateNotFound(t *testing.T) {
	t.Parallel()
	tmpl := GetTemplate("NonExistent")
	if tmpl != nil {
		t.Fatalf("should return nil for non-existent template")
	}
}

func TestListTemplates(t *testing.T) {
	t.Parallel()
	list := ListTemplates()
	if len(list) < 3 {
		t.Fatalf("ListTemplates len = %d, want >= 3", len(list))
	}
}
