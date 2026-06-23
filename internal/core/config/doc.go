// Package config loads and saves PortWeave's TOML application configuration.
//
// Load returns a defaulted Schema when the file is missing (not an error);
// parse failures are wrapped as invalid-input errors. Save writes atomically
// via a temp file + rename.
package config
