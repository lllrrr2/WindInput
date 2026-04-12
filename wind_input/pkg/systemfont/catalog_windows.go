//go:build windows

package systemfont

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"
)

// FontInfo describes a system-installed font family.
type FontInfo struct {
	Family      string `json:"family"`
	DisplayName string `json:"display_name"`
}

type catalog struct {
	fonts        []FontInfo
	families     map[string][]string
	displayNames map[string]string
}

var (
	catalogOnce sync.Once
	cached      catalog
	cachedErr   error
)

var styleSuffixes = []string{
	" ExtraBold",
	" Extra Light",
	" ExtraLight",
	" SemiBold",
	" Semibold",
	" Semi Light",
	" SemiLight",
	" Medium",
	" Regular",
	" Italic",
	" Oblique",
	" Condensed",
	" Narrow",
	" Light",
	" Black",
	" Bold",
	" Thin",
}

var fallbackFamilies = []FontInfo{
	{Family: "Microsoft YaHei", DisplayName: "Microsoft YaHei"},
	{Family: "Segoe UI", DisplayName: "Segoe UI"},
	{Family: "Segoe UI Symbol", DisplayName: "Segoe UI Symbol"},
	{Family: "Arial", DisplayName: "Arial"},
}

func systemFontsDir() string {
	winDir := os.Getenv("WINDIR")
	if winDir == "" {
		winDir = os.Getenv("SystemRoot")
	}
	if winDir == "" {
		winDir = "C:\\Windows"
	}
	return filepath.Join(winDir, "Fonts")
}

func normalizeKey(v string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(v)), " "))
}

func trimRegistrySuffix(name string) string {
	name = strings.TrimSpace(name)
	for _, suffix := range []string{" (TrueType)", " (OpenType)", " (All res)"} {
		if strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix)) {
			name = strings.TrimSpace(name[:len(name)-len(suffix)])
			break
		}
	}
	return name
}

func trimStyleSuffix(name string) string {
	name = strings.TrimSpace(name)
	for {
		trimmed := name
		lower := strings.ToLower(name)
		for _, suffix := range styleSuffixes {
			if strings.HasSuffix(lower, strings.ToLower(suffix)) {
				trimmed = strings.TrimSpace(name[:len(name)-len(suffix)])
				break
			}
		}
		if trimmed == name {
			return name
		}
		name = trimmed
	}
}

func extractFamilies(displayName string) []string {
	raw := trimRegistrySuffix(displayName)
	if raw == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var out []string

	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := normalizeKey(name)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}

	for _, part := range strings.Split(raw, "&") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		add(trimStyleSuffix(part))
	}

	if len(out) == 0 {
		add(trimStyleSuffix(raw))
	}

	return out
}

func resolveFontPath(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}
	if filepath.IsAbs(fileName) {
		return fileName
	}
	return filepath.Join(systemFontsDir(), fileName)
}

func appendUniquePath(paths []string, path string) []string {
	key := normalizeKey(path)
	for _, existing := range paths {
		if normalizeKey(existing) == key {
			return paths
		}
	}
	return append(paths, path)
}

func loadRegistryFonts(root registry.Key, path string, cat *catalog) error {
	key, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		return err
	}
	defer key.Close()

	names, err := key.ReadValueNames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		fontPath := resolveFontPath(value)
		for _, family := range extractFamilies(name) {
			fkey := normalizeKey(family)
			cat.families[fkey] = appendUniquePath(cat.families[fkey], fontPath)
			if _, ok := cat.displayNames[fkey]; !ok {
				cat.displayNames[fkey] = family
			}
		}
		raw := trimRegistrySuffix(name)
		for _, alias := range strings.Split(raw, "&") {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			akey := normalizeKey(alias)
			cat.families[akey] = appendUniquePath(cat.families[akey], fontPath)
		}
	}

	return nil
}

func ensureCatalog() error {
	catalogOnce.Do(func() {
		cached = catalog{
			families:     make(map[string][]string),
			displayNames: make(map[string]string),
		}

		paths := []struct {
			root registry.Key
			path string
		}{
			{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`},
			{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`},
		}

		for _, item := range paths {
			_ = loadRegistryFonts(item.root, item.path, &cached)
		}

		seen := make(map[string]struct{})
		for key, paths := range cached.families {
			if len(paths) == 0 {
				continue
			}
			name := cached.displayNames[key]
			if name == "" {
				continue
			}
			seen[key] = struct{}{}
			cached.fonts = append(cached.fonts, FontInfo{
				Family:      name,
				DisplayName: name,
			})
		}

		if len(cached.fonts) == 0 {
			for _, fallback := range fallbackFamilies {
				key := normalizeKey(fallback.Family)
				if _, ok := seen[key]; ok {
					continue
				}
				cached.displayNames[key] = fallback.Family
				cached.fonts = append(cached.fonts, fallback)
			}
		}

		sort.Slice(cached.fonts, func(i, j int) bool {
			return strings.ToLower(cached.fonts[i].DisplayName) < strings.ToLower(cached.fonts[j].DisplayName)
		})
	})
	return cachedErr
}

// List returns installed system font families.
func List() ([]FontInfo, error) {
	err := ensureCatalog()
	fonts := make([]FontInfo, len(cached.fonts))
	copy(fonts, cached.fonts)
	return fonts, err
}

// HasFamily reports whether the family exists in the catalog.
func HasFamily(family string) bool {
	_ = ensureCatalog()
	_, ok := cached.families[normalizeKey(family)]
	return ok
}

func isSingleFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf":
		return true
	default:
		return false
	}
}

// ResolveFile returns a file path for a family name.
// When singleFontOnly is true, TTC collections are skipped.
func ResolveFile(family string, singleFontOnly bool) string {
	_ = ensureCatalog()
	paths := cached.families[normalizeKey(family)]
	for _, path := range paths {
		if singleFontOnly && !isSingleFontFile(path) {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
