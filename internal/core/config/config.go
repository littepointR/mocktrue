package config

import (
	"fmt"
	"os"
	"path/filepath"

	mterrors "github.com/littepointR/mocktrue/internal/core/errors"
	"github.com/pelletier/go-toml/v2"
)

// Schema is the application configuration model.
type Schema struct {
	App     AppConfig     `toml:"app"`
	MCP     MCPConfig     `toml:"mcp"`
	Window  WindowConfig  `toml:"window"`
	Serial  SerialConfig  `toml:"serial"`
	Modules ModulesConfig `toml:"modules"`
}

// AppConfig holds application identity metadata.
type AppConfig struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

// MCPConfig controls the local Model Context Protocol server.
type MCPConfig struct {
	Enabled           bool   `toml:"enabled"`
	Host              string `toml:"host"`
	Port              int    `toml:"port"`
	Path              string `toml:"path"`
	AllowLocalOrigins bool   `toml:"allow_local_origins"`
}

// WindowConfig holds persisted window defaults.
type WindowConfig struct {
	Width  int    `toml:"width"`
	Height int    `toml:"height"`
	Theme  string `toml:"theme"` // "dark" | "light"
}

// SerialConfig holds serial module configuration.
type SerialConfig struct {
	Presets      []PortPreset  `toml:"presets"`
	QuickButtons []QuickButton `toml:"quick_buttons"`
}

// PortPreset represents a saved port configuration.
type PortPreset struct {
	Name     string `toml:"name"`
	Port     string `toml:"port"`
	BaudRate int    `toml:"baud_rate"`
	DataBits int    `toml:"data_bits"`
	StopBits string `toml:"stop_bits"`
	Parity   string `toml:"parity"`
	FlowMode string `toml:"flow_mode"`
}

// QuickButton represents a preset send button.
type QuickButton struct {
	ID      string `toml:"id"`
	Label   string `toml:"label"`
	Content string `toml:"content"`
	Mode    string `toml:"mode"` // "ascii" | "hex"
}

// ModulesConfig holds per-module sub-configuration as raw maps.
type ModulesConfig struct {
	Serial map[string]any `toml:"serial"`
}

// Default returns the built-in default configuration.
func Default() *Schema {
	return &Schema{
		App: AppConfig{Name: "PortWeave", Version: "0.1.0"},
		MCP: MCPConfig{
			Enabled:           true,
			Host:              "127.0.0.1",
			Port:              39391,
			Path:              "/mcp",
			AllowLocalOrigins: true,
		},
		Window: WindowConfig{Width: 1280, Height: 800, Theme: "dark"},
		Serial: SerialConfig{
			Presets: []PortPreset{
				{Name: "默认 115200", BaudRate: 115200, DataBits: 8, StopBits: "1", Parity: "none", FlowMode: "none"},
			},
			QuickButtons: []QuickButton{},
		},
		Modules: ModulesConfig{
			Serial: map[string]any{},
		},
	}
}

// Load reads and parses the TOML file at path. A missing file is not an
// error: Default is returned. Parse failures are wrapped as invalid input.
func Load(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, mterrors.Wrap(mterrors.CodeIO, "read config file", err)
	}

	cfg := Default()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, mterrors.Wrap(mterrors.CodeInvalid, "parse config TOML", err)
	}
	return cfg, nil
}

// Save writes the schema to path atomically (temp file + rename) so a crash
// never leaves a half-written config. An empty path is rejected as invalid
// input.
func (s *Schema) Save(path string) error {
	if path == "" {
		return mterrors.New(mterrors.CodeInvalid, "config path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return mterrors.Wrap(mterrors.CodePlatform, "create config directory", err)
	}

	data, err := toml.Marshal(s)
	if err != nil {
		return mterrors.Wrap(mterrors.CodeInternal, "marshal config TOML", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return mterrors.Wrap(mterrors.CodeIO, "create temp config", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return mterrors.Wrap(mterrors.CodeIO, "write temp config", err)
	}
	if err := tmp.Close(); err != nil {
		return mterrors.Wrap(mterrors.CodeIO, "close temp config", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return mterrors.Wrap(mterrors.CodeIO, fmt.Sprintf("rename temp config to %s", path), err)
	}
	return nil
}
