//go:build !windows

package virtualserial

import "testing"

func TestDefaultBackendHasName(t *testing.T) {
	if DefaultBackend().Name() == "" {
		t.Fatalf("DefaultBackend name must not be empty")
	}
}
