package port

import (
	"context"
	"testing"
)

func TestEnumerateReturnsSlice(t *testing.T) {
	t.Parallel()
	ports, err := Enumerate(context.Background())
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	// ports may be empty on CI (no hardware), but must be a valid slice.
	if ports == nil {
		t.Fatalf("Enumerate must not return nil slice")
	}
}

func TestPortInfoFieldsPopulated(t *testing.T) {
	t.Parallel()
	ports, err := Enumerate(context.Background())
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	for _, p := range ports {
		if p.Name == "" {
			t.Fatalf("PortInfo.Name must not be empty: %+v", p)
		}
		// USB ports typically have Vendor/Product, but some drivers omit them.
		// Just verify IsUSB is consistent: a product without a vendor would be suspect.
		if p.IsUSB && p.VID == "" && p.PID != "" {
			t.Fatalf("USB PortInfo has PID without VID: %+v", p)
		}
	}
}

func TestListPortsReturnsEnumeratedNames(t *testing.T) {
	t.Parallel()
	ports, err := Enumerate(context.Background())
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	names, err := ListPorts(context.Background())
	if err != nil {
		t.Fatalf("ListPorts failed: %v", err)
	}
	if len(names) != len(ports) {
		t.Fatalf("ListPorts len = %d, want %d", len(names), len(ports))
	}
	for i := range ports {
		if names[i] != ports[i].Name {
			t.Fatalf("ListPorts[%d] = %q, want %q", i, names[i], ports[i].Name)
		}
	}
}
