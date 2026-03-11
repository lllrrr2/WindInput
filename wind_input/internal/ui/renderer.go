package ui

import (
	"image"
	"image/color"
	"os"
	"sync"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/huanfeng/wind_input/pkg/theme"
	"golang.org/x/image/font"
)

// RenderConfig contains rendering configuration
type RenderConfig struct {
	FontPath        string
	FontSize        float64
	IndexFontSize   float64
	Padding         float64
	ItemHeight      float64
	CornerRadius    float64
	BackgroundColor color.Color
	TextColor       color.Color
	IndexColor      color.Color
	IndexBgColor    color.Color
	InputBgColor    color.Color
	InputTextColor  color.Color
	BorderColor     color.Color
	HoverBgColor    color.Color // Background color for hovered candidate
	Layout          string      // "horizontal" or "vertical"
	HidePreedit     bool        // Hide preedit area when inline_preedit is enabled
	IndexStyle      string      // "circle" (default) or "text" (plain text index)
	AccentBarColor  color.Color // Left accent bar color, nil = no bar
	HasAccentBar    bool        // Whether to draw accent bar
}

// DefaultRenderConfig returns default rendering configuration with DPI scaling
func DefaultRenderConfig() RenderConfig {
	// Get DPI scale factor
	scale := GetDPIScale()

	return RenderConfig{
		FontPath:        "", // Will use system font
		FontSize:        18 * scale,
		IndexFontSize:   14 * scale,
		Padding:         10 * scale,
		ItemHeight:      32 * scale,
		CornerRadius:    8 * scale,
		BackgroundColor: color.RGBA{255, 255, 255, 245}, // Slightly transparent white
		TextColor:       color.RGBA{30, 30, 30, 255},
		IndexColor:      color.RGBA{255, 255, 255, 255},
		IndexBgColor:    color.RGBA{66, 133, 244, 255}, // Blue
		InputBgColor:    color.RGBA{240, 240, 240, 255},
		InputTextColor:  color.RGBA{100, 100, 100, 255},
		BorderColor:     color.RGBA{200, 200, 200, 255},
		HoverBgColor:    color.RGBA{230, 240, 255, 255}, // Light blue for hover
		Layout:          "horizontal",                   // Default to horizontal layout
		HidePreedit:     false,
	}
}

// fontCache caches loaded fonts and font faces
type fontCache struct {
	mu       sync.RWMutex
	font     *truetype.Font
	fontPath string
	faces    map[float64]font.Face // Cache font faces by size
}

func newFontCache() *fontCache {
	return &fontCache{
		faces: make(map[float64]font.Face),
	}
}

// loadFont loads a TTF font from the given path
func (fc *fontCache) loadFont(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f, err := truetype.Parse(data)
	if err != nil {
		return err
	}
	fc.font = f
	fc.fontPath = path
	fc.faces = make(map[float64]font.Face) // Clear cached faces
	return nil
}

// getFace returns a cached font face for the given size
func (fc *fontCache) getFace(size float64) font.Face {
	fc.mu.RLock()
	if face, ok := fc.faces[size]; ok {
		fc.mu.RUnlock()
		return face
	}
	fc.mu.RUnlock()

	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Double-check
	if face, ok := fc.faces[size]; ok {
		return face
	}

	if fc.font == nil {
		return nil
	}

	face := truetype.NewFace(fc.font, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	fc.faces[size] = face
	return face
}

// Renderer renders candidate window content
type Renderer struct {
	config        RenderConfig
	fontPath      string
	fontCache     *fontCache
	fontReady     bool
	resolvedTheme *theme.ResolvedTheme
}

// NewRenderer creates a new renderer
func NewRenderer(config RenderConfig) *Renderer {
	r := &Renderer{
		config:    config,
		fontCache: newFontCache(),
	}
	// Pre-load font on creation
	r.ensureFontLoaded()
	return r
}

// SetFontPath sets the font path
func (r *Renderer) SetFontPath(path string) {
	r.fontPath = path
	r.fontReady = false
	r.ensureFontLoaded()
}

// UpdateFont updates font settings
func (r *Renderer) UpdateFont(fontSize float64, fontPath string) {
	scale := GetDPIScale()

	if fontSize > 0 {
		r.config.FontSize = fontSize * scale
		r.config.IndexFontSize = (fontSize - 4) * scale
	}

	if fontPath != "" && fontPath != r.fontPath {
		r.fontPath = fontPath
		r.fontReady = false
		r.ensureFontLoaded()
	}
}

// SetLayout sets the candidate layout mode
func (r *Renderer) SetLayout(layout string) {
	if layout == "horizontal" || layout == "vertical" {
		r.config.Layout = layout
	}
}

// SetHidePreedit sets whether to hide the preedit area
func (r *Renderer) SetHidePreedit(hide bool) {
	r.config.HidePreedit = hide
}

// SetTheme sets the theme for the renderer and updates colors
func (r *Renderer) SetTheme(resolved *theme.ResolvedTheme) {
	if resolved == nil {
		return
	}
	r.resolvedTheme = resolved
	// Update config colors from theme
	colors := resolved.CandidateWindow
	r.config.BackgroundColor = colors.BackgroundColor
	r.config.BorderColor = colors.BorderColor
	r.config.TextColor = colors.TextColor
	r.config.IndexColor = colors.IndexColor
	r.config.IndexBgColor = colors.IndexBgColor
	r.config.HoverBgColor = colors.HoverBgColor
	r.config.InputBgColor = colors.InputBgColor
	r.config.InputTextColor = colors.InputTextColor
	// Update style from theme
	r.config.IndexStyle = resolved.Style.IndexStyle
	r.config.AccentBarColor = resolved.Style.AccentBarColor
	r.config.HasAccentBar = resolved.Style.HasAccentBar
}

// getCommentColor returns the comment color from theme or default
func (r *Renderer) getCommentColor() color.Color {
	if r.resolvedTheme != nil {
		return r.resolvedTheme.CandidateWindow.CommentColor
	}
	return color.RGBA{150, 150, 150, 255}
}

// getShadowColor returns the shadow color from theme or default
func (r *Renderer) getShadowColor() color.Color {
	if r.resolvedTheme != nil {
		return r.resolvedTheme.CandidateWindow.ShadowColor
	}
	return color.RGBA{0, 0, 0, 15}
}

// getModeIndicatorColors returns mode indicator colors from theme or defaults
func (r *Renderer) getModeIndicatorColors() (bgColor, textColor color.Color) {
	if r.resolvedTheme != nil {
		return r.resolvedTheme.ModeIndicator.BackgroundColor, r.resolvedTheme.ModeIndicator.TextColor
	}
	return color.RGBA{50, 50, 50, 230}, color.RGBA{255, 255, 255, 255}
}

// GetLayout returns the current layout mode
func (r *Renderer) GetLayout() string {
	return r.config.Layout
}

// ensureFontLoaded loads the font if not already cached
func (r *Renderer) ensureFontLoaded() {
	if r.fontReady && r.fontCache.font != nil {
		return
	}

	r.fontCache.mu.Lock()
	defer r.fontCache.mu.Unlock()

	// Try user-specified font first
	if r.fontPath != "" {
		if err := r.fontCache.loadFont(r.fontPath); err == nil {
			r.fontReady = true
			return
		}
	}

	// Try common Windows fonts (prefer .ttf over .ttc)
	fonts := []string{
		"C:/Windows/Fonts/simhei.ttf",  // SimHei
		"C:/Windows/Fonts/simsun.ttf",  // SimSun (ttf version)
		"C:/Windows/Fonts/msyh.ttf",    // Microsoft YaHei (ttf version)
		"C:/Windows/Fonts/arial.ttf",   // Arial fallback
		"C:/Windows/Fonts/segoeui.ttf", // Segoe UI
	}
	for _, path := range fonts {
		if err := r.fontCache.loadFont(path); err == nil {
			r.fontPath = path
			r.fontReady = true
			return
		}
	}
}

// drawChevronLeft draws a left-pointing chevron (‹) at the given center position
func (r *Renderer) drawChevronLeft(dc *gg.Context, cx, cy, size, lineWidth float64) {
	halfH := size / 2
	halfW := size * 0.35 // narrower for elegance
	dc.SetLineWidth(lineWidth)
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineJoin(gg.LineJoinRound)
	dc.MoveTo(cx+halfW, cy-halfH)
	dc.LineTo(cx-halfW, cy)
	dc.LineTo(cx+halfW, cy+halfH)
	dc.Stroke()
}

// drawChevronRight draws a right-pointing chevron (›) at the given center position
func (r *Renderer) drawChevronRight(dc *gg.Context, cx, cy, size, lineWidth float64) {
	halfH := size / 2
	halfW := size * 0.35
	dc.SetLineWidth(lineWidth)
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineJoin(gg.LineJoinRound)
	dc.MoveTo(cx-halfW, cy-halfH)
	dc.LineTo(cx+halfW, cy)
	dc.LineTo(cx-halfW, cy+halfH)
	dc.Stroke()
}

func (r *Renderer) drawRoundedRect(dc *gg.Context, x, y, w, h, radius float64) {
	dc.NewSubPath()
	dc.MoveTo(x+radius, y)
	dc.LineTo(x+w-radius, y)
	dc.DrawArc(x+w-radius, y+radius, radius, -gg.Radians(90), 0)
	dc.LineTo(x+w, y+h-radius)
	dc.DrawArc(x+w-radius, y+h-radius, radius, 0, gg.Radians(90))
	dc.LineTo(x+radius, y+h)
	dc.DrawArc(x+radius, y+h-radius, radius, gg.Radians(90), gg.Radians(180))
	dc.LineTo(x, y+radius)
	dc.DrawArc(x+radius, y+radius, radius, gg.Radians(180), gg.Radians(270))
	dc.ClosePath()
}

// RenderModeIndicator renders a mode indicator with adaptive width
func (r *Renderer) RenderModeIndicator(mode string) *image.RGBA {
	scale := GetDPIScale()

	minWidth := 50.0 * scale
	height := 36.0 * scale
	fontSize := 20.0 * scale
	padding := 12.0 * scale

	// Use cached font face
	face := r.fontCache.getFace(fontSize)

	// Measure text width to determine canvas width
	var textWidth float64
	if face != nil {
		tmpDc := gg.NewContext(1, 1)
		tmpDc.SetFontFace(face)
		textWidth, _ = tmpDc.MeasureString(mode)
	}

	// Adaptive width: max(minWidth, textWidth + padding*2)
	width := textWidth + padding*2
	if width < minWidth {
		width = minWidth
	}

	dc := gg.NewContext(int(width), int(height))

	// Get colors from theme
	bgColor, textColor := r.getModeIndicatorColors()

	// Draw background
	dc.SetColor(bgColor)
	r.drawRoundedRect(dc, 2*scale, 2*scale, width-4*scale, height-4*scale, 6*scale)
	dc.Fill()

	if face != nil {
		dc.SetFontFace(face)
	}

	// Draw mode text (centered)
	dc.SetColor(textColor)
	tw, _ := dc.MeasureString(mode)
	dc.DrawString(mode, width/2-tw/2, height/2+7*scale)

	return dc.Image().(*image.RGBA)
}
