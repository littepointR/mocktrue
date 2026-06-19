package serial

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListFontFamiliesFromDirsReturnsSortedUniqueFontNames(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := []string{
		"System Mono.ttf",
		"System Mono.otf",
		"PingFang.ttc",
		"ignored.txt",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("font"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	got := listFontFamiliesFromDirs([]string{dir})
	want := []string{"PingFang", "System Mono"}
	if len(got) != len(want) {
		t.Fatalf("font count = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("font[%d] = %q, want %q; all fonts: %v", i, got[i], want[i], got)
		}
	}
}
