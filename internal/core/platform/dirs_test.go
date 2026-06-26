package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolvePathsRequiresAppName(t *testing.T) {
	t.Parallel()
	if _, err := ResolvePaths(""); err == nil {
		t.Fatalf("ResolvePaths(\"\") must error")
	}
}

func TestResolvePathsCreatesAllDirsAbsoluteUnderIsolatedHome(t *testing.T) {
	// Not parallel: mutates process environment via t.Setenv.
	tmp := isolateUserDirs(t)

	paths, err := ResolvePaths("portweave-test")
	if err != nil {
		t.Fatalf("ResolvePaths failed: %v", err)
	}

	for name, dir := range map[string]string{
		"ConfigDir": paths.ConfigDir,
		"DataDir":   paths.DataDir,
		"CacheDir":  paths.CacheDir,
		"LogDir":    paths.LogDir,
	} {
		if dir == "" {
			t.Fatalf("%s must not be empty", name)
		}
		if !filepath.IsAbs(dir) {
			t.Fatalf("%s must be absolute, got %q", name, dir)
		}
		if !strings.Contains(dir, "portweave-test") {
			t.Fatalf("%s must contain app name, got %q", name, dir)
		}
		if !strings.HasPrefix(dir, tmp) {
			t.Fatalf("%s must live under isolated home %s, got %q", name, tmp, dir)
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			t.Fatalf("%s must exist as a directory, got err=%v", name, err)
		}
	}
}

func isolateUserDirs(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// os.UserConfigDir/UserCacheDir read $HOME on macOS, XDG_* on Linux,
	// and APPDATA/LOCALAPPDATA/USERPROFILE on Windows, so redirect every
	// platform's inputs to keep this test isolated from the real user profile.
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmp, "AppData", "Roaming"))
		t.Setenv("LOCALAPPDATA", filepath.Join(tmp, "AppData", "Local"))
		t.Setenv("USERPROFILE", tmp)
	}
	return tmp
}

func TestEnsureDirCreatesNested(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	target := filepath.Join(base, "a", "b", "c")
	if err := EnsureDir(target, 0o755); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("EnsureDir did not create the directory: %v", err)
	}
}

func TestEnsureDirIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := EnsureDir(dir, 0o755); err != nil {
		t.Fatalf("EnsureDir on existing dir must not error, got %v", err)
	}
}

func TestEnsureDirFailsUnderAFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	// A path whose parent is a regular file cannot become a directory.
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	target := filepath.Join(blocker, "child")
	if err := EnsureDir(target, 0o700); err == nil {
		t.Fatalf("EnsureDir under a file must error")
	}
}
