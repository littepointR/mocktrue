package serial

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ListSystemFonts returns available font family candidates for terminal display.
func (s *Service) ListSystemFonts() []string {
	return listFontFamiliesFromDirs(systemFontDirs())
}

func systemFontDirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return compactDirs([]string{
			"/System/Library/Fonts",
			"/Library/Fonts",
			filepath.Join(home, "Library/Fonts"),
		})
	case "windows":
		return compactDirs([]string{
			filepath.Join(os.Getenv("WINDIR"), "Fonts"),
			filepath.Join(home, "AppData/Local/Microsoft/Windows/Fonts"),
		})
	default:
		return compactDirs([]string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
			filepath.Join(home, ".fonts"),
			filepath.Join(home, ".local/share/fonts"),
		})
	}
}

func compactDirs(dirs []string) []string {
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if strings.TrimSpace(dir) != "" {
			out = append(out, dir)
		}
	}
	return out
}

func listFontFamiliesFromDirs(dirs []string) []string {
	families := make(map[string]struct{})
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			name := fontFamilyName(path)
			if name != "" {
				families[name] = struct{}{}
			}
			return nil
		})
	}

	result := make([]string, 0, len(families))
	for name := range families {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func fontFamilyName(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ttf", ".ttc", ".otf":
	default:
		return ""
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name = strings.TrimSpace(strings.ReplaceAll(name, "_", " "))
	return name
}
