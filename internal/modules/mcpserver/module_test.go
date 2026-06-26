package mcpserver

import "testing"

func TestModuleMetadataAndServices(t *testing.T) {
	module := New(nil)

	if got := module.ID(); got != "mcpserver" {
		t.Fatalf("ID() = %q, want mcpserver", got)
	}

	manifest := module.Manifest()
	if manifest.ID != "mcpserver" {
		t.Fatalf("Manifest().ID = %q, want mcpserver", manifest.ID)
	}
	if manifest.Version != "0.1.0" {
		t.Fatalf("Manifest().Version = %q, want 0.1.0", manifest.Version)
	}
	if len(manifest.Dependencies) != 1 || manifest.Dependencies[0] != "serial" {
		t.Fatalf("Manifest().Dependencies = %#v, want [serial]", manifest.Dependencies)
	}

	if services := module.Services(); services != nil {
		t.Fatalf("Services() = %#v, want nil", services)
	}
	if services := module.ServicesWrapped(); services != nil {
		t.Fatalf("ServicesWrapped() = %#v, want nil", services)
	}

	module.Dispose()
}
