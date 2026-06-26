package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsDefaultNoError(t *testing.T) {
	t.Parallel()
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load of missing file must not error, got %v", err)
	}
	def := Default()
	if cfg.App.Name != def.App.Name {
		t.Fatalf("App.Name = %q, want default %q", cfg.App.Name, def.App.Name)
	}
}

func TestLoadParsesValidTOML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
app = { name = "PortWeave", version = "0.1.0" }
[mcp]
enabled = true
host = "127.0.0.1"
port = 39391
path = "/mcp"
allow_local_origins = true
[window]
width = 1200
height = 800
theme = "dark"
[modules.serial]
baudrate = 115200
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.App.Name != "PortWeave" || cfg.Window.Width != 1200 || cfg.Window.Theme != "dark" {
		t.Fatalf("parsed config wrong: %+v", cfg)
	}
	if !cfg.MCP.Enabled || cfg.MCP.Host != "127.0.0.1" || cfg.MCP.Port != 39391 || cfg.MCP.Path != "/mcp" || !cfg.MCP.AllowLocalOrigins {
		t.Fatalf("parsed mcp config wrong: %+v", cfg.MCP)
	}
	if cfg.Modules.Serial["baudrate"] != int64(115200) {
		t.Fatalf("serial baudrate = %v, want 115200", cfg.Modules.Serial["baudrate"])
	}
}

func TestDefaultIncludesLocalMCPServer(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if !cfg.MCP.Enabled {
		t.Fatalf("default MCP server must be enabled")
	}
	if cfg.MCP.Host != "127.0.0.1" {
		t.Fatalf("default MCP host = %q, want 127.0.0.1", cfg.MCP.Host)
	}
	if cfg.MCP.Port != 39391 {
		t.Fatalf("default MCP port = %d, want 39391", cfg.MCP.Port)
	}
	if cfg.MCP.Path != "/mcp" {
		t.Fatalf("default MCP path = %q, want /mcp", cfg.MCP.Path)
	}
	if !cfg.MCP.AllowLocalOrigins {
		t.Fatalf("default MCP local origin allowance must be true")
	}
}

func TestLoadInvalidTOMLReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("not = valid = toml = ="), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("Load of invalid TOML must error")
	}
}

func TestLoadReadErrorReturnsError(t *testing.T) {
	t.Parallel()
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatalf("Load of a directory path must return a read error")
	}
}

func TestSaveThenLoadRoundtrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := Default()
	cfg.Window.Width = 1600
	cfg.Window.Height = 900
	cfg.Window.Theme = "dark"

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}
	if loaded.Window.Width != 1600 || loaded.Window.Height != 900 || loaded.Window.Theme != "dark" {
		t.Fatalf("roundtrip lost values: %+v", loaded.Window)
	}
}

func TestSaveEmptyPathRejected(t *testing.T) {
	t.Parallel()
	if err := Default().Save(""); err == nil {
		t.Fatalf("Save(\"\") must error")
	}
}

func TestSaveCreateDirectoryErrorReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}
	if err := Default().Save(filepath.Join(blocker, "config.toml")); err == nil {
		t.Fatalf("Save under a file path must return a directory creation error")
	}
}

func TestSaveIsAtomicNoTempResidue(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := Default().Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "config.toml" {
			t.Fatalf("unexpected leftover file: %s", e.Name())
		}
	}
}
