package ui

import "os"

// GDI font weight constants (Windows LOGFONT.lfWeight values)
const (
	FontWeightThin       = 100
	FontWeightExtraLight = 200
	FontWeightLight      = 300
	FontWeightNormal     = 400 // Default
	FontWeightMedium     = 500
	FontWeightSemiBold   = 600
	FontWeightBold       = 700
)

// FontConfig holds centralized font configuration for all UI components.
// Instead of each component maintaining its own hardcoded font list,
// all components share this configuration for consistent font management.
type FontConfig struct {
	// PrimaryFont is the user-configured font file path (may be empty for auto-detection)
	PrimaryFont string
	// SystemFonts lists system fonts in priority order for fallback.
	// When a font lacks certain glyphs, subsequent fonts in the list are tried.
	SystemFonts []string

	// GDIFontWeight controls the font weight for GDI rendering.
	// Valid range: 100 (thin) to 900 (heavy). Common values:
	//   400 = Normal (default), 500 = Medium, 600 = SemiBold, 700 = Bold
	// Higher values produce thicker/heavier strokes.
	GDIFontWeight int

	// GDIFontScale controls the font size multiplier for GDI rendering.
	// Default 1.0 means lfHeight = -fontSize (character height = fontSize pixels).
	// Values > 1.0 produce larger text (e.g., 1.15 makes GDI text ~15% larger).
	// Useful for matching visual size between GDI and FreeType backends.
	GDIFontScale float64
}

// defaultSystemFonts is the default system font fallback chain for Windows.
// Fonts are ordered by priority: CJK-capable fonts first, then symbol/Latin fonts.
var defaultSystemFonts = []string{
	"C:/Windows/Fonts/msyh.ttc",    // Microsoft YaHei (best CJK + Latin coverage)
	"C:/Windows/Fonts/simhei.ttf",  // SimHei (CJK)
	"C:/Windows/Fonts/simsun.ttc",  // SimSun (CJK)
	"C:/Windows/Fonts/segoeui.ttf", // Segoe UI (Latin, UI symbols)
	"C:/Windows/Fonts/arial.ttf",   // Arial (Latin fallback)
}

// NewFontConfig creates a FontConfig with the default system font chain.
func NewFontConfig() *FontConfig {
	return &FontConfig{
		SystemFonts:   append([]string{}, defaultSystemFonts...),
		GDIFontWeight: FontWeightMedium,
		GDIFontScale:  1.0,
	}
}

// ResolvePrimaryFont returns the first available font path.
// If PrimaryFont is set and exists, it is used; otherwise the SystemFonts
// chain is searched in order.
func (fc *FontConfig) ResolvePrimaryFont() string {
	if fc.PrimaryFont != "" {
		if _, err := os.Stat(fc.PrimaryFont); err == nil {
			return fc.PrimaryFont
		}
	}
	for _, path := range fc.SystemFonts {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// GetFallbackFonts returns all available fonts after the primary,
// in priority order, for fallback rendering of missing glyphs.
func (fc *FontConfig) GetFallbackFonts() []string {
	primary := fc.ResolvePrimaryFont()
	var fallbacks []string
	for _, path := range fc.SystemFonts {
		if path != primary {
			if _, err := os.Stat(path); err == nil {
				fallbacks = append(fallbacks, path)
			}
		}
	}
	return fallbacks
}

// SetPrimaryFont sets a user-configured primary font.
func (fc *FontConfig) SetPrimaryFont(fontPath string) {
	fc.PrimaryFont = fontPath
}

// SetGDIFontWeight sets the GDI font weight (100-900).
// Common values: 400=Normal, 500=Medium, 600=SemiBold, 700=Bold.
func (fc *FontConfig) SetGDIFontWeight(weight int) {
	if weight < 100 {
		weight = 100
	}
	if weight > 900 {
		weight = 900
	}
	fc.GDIFontWeight = weight
}

// SetGDIFontScale sets the GDI font size multiplier (0.5-2.0).
func (fc *FontConfig) SetGDIFontScale(scale float64) {
	if scale < 0.5 {
		scale = 0.5
	}
	if scale > 2.0 {
		scale = 2.0
	}
	fc.GDIFontScale = scale
}

// GetEffectiveGDIWeight returns the GDI font weight, defaulting to 400 if unset.
func (fc *FontConfig) GetEffectiveGDIWeight() int {
	if fc.GDIFontWeight <= 0 {
		return FontWeightNormal
	}
	return fc.GDIFontWeight
}

// GetEffectiveGDIScale returns the GDI font scale, defaulting to 1.0 if unset.
func (fc *FontConfig) GetEffectiveGDIScale() float64 {
	if fc.GDIFontScale <= 0 {
		return 1.0
	}
	return fc.GDIFontScale
}
