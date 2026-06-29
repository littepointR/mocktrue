//go:build linux || darwin || freebsd || netbsd || openbsd

package virtualserial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreerrors "github.com/littepointR/portweave/internal/core/errors"
)

func TestSocatEndpointPathsRejectUnsafeNamesWithoutTouchingFiles(t *testing.T) {
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write victim: %v", err)
	}

	for _, tc := range []struct {
		name string
		port string
	}{
		{name: "absolute", port: victim},
		{name: "traversal", port: "../victim"},
		{name: "dotdot", port: "tty..bad"},
		{name: "nested", port: "nested/name"},
		{name: "backslash", port: `nested\name`},
		{name: "shell metacharacter", port: "tty;rm"},
		{name: "space", port: "tty bad"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			port1, port2, dir, err := socatEndpointPaths(tc.port, "safe")
			if err == nil {
				if dir != "" {
					_ = os.RemoveAll(dir)
				}
				t.Fatalf("socatEndpointPaths(%q) = %q/%q in %q, want error", tc.port, port1, port2, dir)
			}
			if code := coreerrors.AsCode(err); code != coreerrors.CodeInvalid {
				t.Fatalf("error code = %v, want invalid: %v", code, err)
			}
			content, readErr := os.ReadFile(victim)
			if readErr != nil {
				t.Fatalf("victim was removed or unreadable: %v", readErr)
			}
			if string(content) != "keep" {
				t.Fatalf("victim content = %q, want keep", string(content))
			}
		})
	}
}

func TestSocatEndpointPathsUseOwnedTempDirectory(t *testing.T) {
	port1, port2, dir, err := socatEndpointPaths("ttyA", "ttyB")
	if err != nil {
		t.Fatalf("socatEndpointPaths: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat temp dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("temp path %q is not a directory", dir)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("temp dir permissions = %v, want 0700", info.Mode().Perm())
	}
	if !strings.HasPrefix(filepath.Base(dir), "portweave-virtualserial-") {
		t.Fatalf("temp dir base = %q, want PortWeave-owned prefix", filepath.Base(dir))
	}
	if filepath.Dir(port1) != dir || filepath.Dir(port2) != dir {
		t.Fatalf("endpoints = %q/%q, want both under %q", port1, port2, dir)
	}
	if filepath.Base(port1) != "ttyA" || filepath.Base(port2) != "ttyB" {
		t.Fatalf("endpoint bases = %q/%q, want ttyA/ttyB", filepath.Base(port1), filepath.Base(port2))
	}
}

func TestSocatPairHandleStopRemovesOwnedTempDirectory(t *testing.T) {
	dir := t.TempDir()
	port1 := filepath.Join(dir, "ttyA")
	port2 := filepath.Join(dir, "ttyB")
	if err := os.WriteFile(port1, []byte("a"), 0o600); err != nil {
		t.Fatalf("write port1: %v", err)
	}
	if err := os.WriteFile(port2, []byte("b"), 0o600); err != nil {
		t.Fatalf("write port2: %v", err)
	}

	if err := (socatPairHandle{paths: []string{port1, port2}, dir: dir}).Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temp dir still exists or stat failed with unexpected error: %v", err)
	}
}
