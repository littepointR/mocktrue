package templates

import (
	"strings"
	"testing"

	"github.com/littepointR/portweave/internal/modules/serial/protocol"
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

func TestProtocolTemplateCatalogDocumentsCurrentKinds(t *testing.T) {
	t.Parallel()
	wantKinds := map[string]protocol.Kind{
		"Modbus RTU": protocol.KindVisual,
		"AA55 自定义帧":  protocol.KindVisual,
		"NMEA":       protocol.KindScript,
	}

	list := ListTemplates()
	if len(list) != len(wantKinds) {
		t.Fatalf("ListTemplates len = %d, want exactly current documented catalog size %d", len(list), len(wantKinds))
	}
	seen := make(map[string]protocol.Kind, len(list))
	for _, item := range list {
		seen[item.Name] = item.Kind
	}
	for name, wantKind := range wantKinds {
		if gotKind, ok := seen[name]; !ok || gotKind != wantKind {
			t.Fatalf("ListTemplates[%q] kind = %q, %v; want %q, true", name, gotKind, ok, wantKind)
		}

		tmpl := GetTemplate(name)
		if tmpl == nil {
			t.Fatalf("GetTemplate(%q) = nil", name)
		}
		if tmpl.Kind != tmpl.Config.Kind {
			t.Fatalf("template %q Kind = %q but Config.Kind = %q", name, tmpl.Kind, tmpl.Config.Kind)
		}
		if tmpl.Kind != wantKind {
			t.Fatalf("template %q Kind = %q, want %q", name, tmpl.Kind, wantKind)
		}
	}

	aa55 := GetTemplate("AA55 自定义帧")
	if aa55.Config.Visual == nil || string(aa55.Config.Visual.Header) != string([]byte{0xaa, 0x55}) || aa55.Config.Visual.Checksum == nil || aa55.Config.Visual.Checksum.Type != "sum8" {
		t.Fatalf("AA55 visual config = %#v, want AA55 header with sum8 checksum", aa55.Config.Visual)
	}
	modbus := GetTemplate("Modbus RTU")
	if modbus.Config.Visual == nil || modbus.Config.Visual.Checksum == nil || modbus.Config.Visual.Checksum.Type != "crc16" {
		t.Fatalf("Modbus visual config = %#v, want crc16 checksum", modbus.Config.Visual)
	}
	nmea := GetTemplate("NMEA")
	if nmea.Config.Script == "" || !strings.Contains(nmea.Config.Script, "len()") || !strings.Contains(nmea.Config.Script, "byte(") {
		t.Fatalf("NMEA script = %q, want protocol parser script helpers len()/byte(i)", nmea.Config.Script)
	}
	if strings.Contains(nmea.Config.Script, "input.") || strings.Contains(nmea.Config.Script, "output.") {
		t.Fatalf("NMEA protocol parser script must not use graph script node input/output APIs: %q", nmea.Config.Script)
	}
}
