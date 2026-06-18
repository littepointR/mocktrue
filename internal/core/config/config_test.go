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
app = { name = "MockTrue", version = "0.1.0" }
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
	if cfg.App.Name != "MockTrue" || cfg.Window.Width != 1200 || cfg.Window.Theme != "dark" {
		t.Fatalf("parsed config wrong: %+v", cfg)
	}
	if cfg.Modules.Serial["baudrate"] != int64(115200) {
		t.Fatalf("serial baudrate = %v, want 115200", cfg.Modules.Serial["baudrate"])
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
