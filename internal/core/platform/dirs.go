package platform

import (
	"os"
	"path/filepath"

	mterrors "github.com/littepointR/portweave/internal/core/errors"
)

// Paths is the set of per-OS application directories, all absolute and
// existing. Immutable once resolved.
type Paths struct {
	ConfigDir string // user configuration directory
	DataDir   string // persistent application data
	CacheDir  string // non-essential cache
	LogDir    string // log files (lives under DataDir)
}

// ResolvePaths resolves platform-appropriate directories for the named app
// and ensures they exist. An empty appName is rejected as invalid input.
//
// It uses the standard library's per-OS conventions:
//   - config/data: os.UserConfigDir() ($XDG_CONFIG_HOME on Linux,
//     %APPDATA% on Windows, $HOME/Library/Application Support on macOS)
//   - cache: os.UserCacheDir() ($XDG_CACHE_HOME / %LOCALAPPDATA% /
//     $HOME/Library/Caches)
//
// Reading env at call time lets tests isolate the home directory.
func ResolvePaths(appName string) (*Paths, error) {
	if appName == "" {
		return nil, mterrors.New(mterrors.CodeInvalid, "appName must not be empty")
	}

	configHome, err := os.UserConfigDir()
	if err != nil {
		return nil, mterrors.Wrap(mterrors.CodePlatform, "resolve config home", err)
	}
	cacheHome, err := os.UserCacheDir()
	if err != nil {
		return nil, mterrors.Wrap(mterrors.CodePlatform, "resolve cache home", err)
	}

	configDir := filepath.Join(configHome, appName)
	dataDir := filepath.Join(configHome, appName) // data colocated with config home
	cacheDir := filepath.Join(cacheHome, appName)
	logDir := filepath.Join(dataDir, "logs")

	dirs := []struct {
		path string
		perm os.FileMode
	}{
		{configDir, 0o700},
		{dataDir, 0o700},
		{cacheDir, 0o755},
		{logDir, 0o755},
	}
	for _, d := range dirs {
		if err := EnsureDir(d.path, d.perm); err != nil {
			return nil, mterrors.Wrap(mterrors.CodePlatform, "ensure application directory", err)
		}
	}

	return &Paths{
		ConfigDir: configDir,
		DataDir:   dataDir,
		CacheDir:  cacheDir,
		LogDir:    logDir,
	}, nil
}

// EnsureDir creates path (and any missing parents) with the given permission.
// It is idempotent: an existing directory is a no-op.
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
