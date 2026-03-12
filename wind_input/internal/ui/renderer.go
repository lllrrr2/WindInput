package ui

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/gogpu/gg"
	ggtext "github.com/gogpu/gg/text"
	"github.com/huanfeng/wind_input/pkg/theme"
)

// RenderConfig contains rendering configuration.
// 这里只描述候选窗的视觉参数；字体文件选择与 fallback 细节由 FontConfig 接管。
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
	HoverBgColor    color.Color    // Background color for hovered candidate
	Layout          string         // "horizontal" or "vertical"
	HidePreedit     bool           // Hide preedit area when inline_preedit is enabled
	IndexStyle      string         // "circle" (default) or "text" (plain text index)
	AccentBarColor  color.Color    // Left accent bar color, nil = no bar
	HasAccentBar    bool           // Whether to draw accent bar
	TextRenderMode  TextRenderMode // "gdi" (Windows native) or "freetype" (original)
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
		BackgroundColor: color.RGBA{255, 255, 255, 255}, // Opaque white
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

// fontCache caches loaded gg/text FontSource instances and per-size faces.
// The shared FontSource is global; this struct only tracks the small face cache
// that varies by requested font size inside one renderer.
// maxFontFaces limits the number of cached ggtext.Face instances per fontCache.
// When exceeded, the least recently used face is closed and evicted.
const maxFontFaces = 16

type fontCache struct {
	mu        sync.RWMutex
	source    *ggtext.FontSource
	fontPath  string
	faces     map[float64]ggtext.Face // Cache font faces by size
	faceOrder []float64               // LRU order: most recently used at end
}

// newFontCache creates an empty per-renderer face cache.
func newFontCache() *fontCache {
	return &fontCache{
		faces: make(map[float64]ggtext.Face),
	}
}

// loadFont records the font path for lazy loading.
func (fc *fontCache) loadFont(path string) error {
	if fc.fontPath == path && fc.source != nil {
		return nil
	}
	// Switching fonts invalidates all per-size faces because gg/text Face objects
	// are derived from the FontSource and size together.
	fc.faces = make(map[float64]ggtext.Face)
	fc.faceOrder = nil
	fc.source = nil
	fc.fontPath = path
	return nil
}

// ensureFontSource loads the gg/text FontSource from the global registry on demand.
// Must be called with fc.mu held for writing.
func (fc *fontCache) ensureFontSource() error {
	if fc.source != nil {
		return nil
	}
	if fc.fontPath == "" {
		return nil
	}
	source, err := GetSharedFontSource(fc.fontPath)
	if err != nil {
		return err
	}
	fc.source = source
	return nil
}

// getFace returns a cached gg/text face for the given size, with LRU eviction.
func (fc *fontCache) getFace(size float64) ggtext.Face {
	fc.mu.RLock()
	if face, ok := fc.faces[size]; ok {
		fc.mu.RUnlock()
		fc.mu.Lock()
		fc.touchLRU(size)
		fc.mu.Unlock()
		return face
	}
	fc.mu.RUnlock()

	fc.mu.Lock()
	defer fc.mu.Unlock()

	if face, ok := fc.faces[size]; ok {
		fc.touchLRU(size)
		return face
	}

	if err := fc.ensureFontSource(); err != nil || fc.source == nil {
		return nil
	}

	// gg/text Face is lightweight, so creating it lazily and caching by size keeps
	// repeated measurements and draws cheap without duplicating font file data.
	face := fc.source.Face(size)

	if len(fc.faces) >= maxFontFaces && len(fc.faceOrder) > 0 {
		oldest := fc.faceOrder[0]
		fc.faceOrder = fc.faceOrder[1:]
		if _, ok := fc.faces[oldest]; ok {
			delete(fc.faces, oldest)
		}
	}

	fc.faces[size] = face
	fc.faceOrder = append(fc.faceOrder, size)
	return face
}

// touchLRU moves size to the end of the LRU order. Must be called with fc.mu held.
func (fc *fontCache) touchLRU(size float64) {
	for i, s := range fc.faceOrder {
		if s == size {
			fc.faceOrder = append(fc.faceOrder[:i], fc.faceOrder[i+1:]...)
			fc.faceOrder = append(fc.faceOrder, size)
			return
		}
	}
	fc.faceOrder = append(fc.faceOrder, size)
}

// Close releases per-instance face references. FontSource instances are shared
// globally and intentionally stay alive for the process lifetime.
func (fc *fontCache) Close() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.faces = make(map[float64]ggtext.Face)
	fc.faceOrder = nil
	fc.source = nil
}

// Renderer renders candidate window content
type Renderer struct {
	config         RenderConfig
	fontPath       string
	fontCache      *fontCache
	fontReady      bool
	resolvedTheme  *theme.ResolvedTheme
	textRenderer   *TextRenderer   // GDI text renderer for native Windows text quality
	dwriteRenderer *DWriteRenderer // DirectWrite text renderer
	textDrawer     TextDrawer      // Active text drawing backend (GDI, FreeType, or DirectWrite)
	fontConfig     *FontConfig     // Centralized font configuration
}

// NewRenderer creates a new renderer
func NewRenderer(config RenderConfig) *Renderer {
	fontCfg := NewFontConfig()

	r := &Renderer{
		config:     config,
		fontConfig: fontCfg,
	}
	r.updateTextDrawer()
	return r
}

func (r *Renderer) resolvePrimaryFontPath() string {
	if r.fontPath != "" {
		r.fontConfig.SetPrimaryFont(r.fontPath)
	}
	// 原生文本后端允许继续落到 TTC 主字体，所以这里走 general path。
	resolved := r.fontConfig.ResolvePrimaryFont()
	if resolved != "" {
		r.fontPath = resolved
	}
	return resolved
}

func (r *Renderer) ensureTextRenderer() *TextRenderer {
	if r.textRenderer != nil {
		return r.textRenderer
	}
	tr := NewTextRenderer()
	tr.SetGDIParams(r.fontConfig.GetEffectiveGDIWeight(), r.fontConfig.GetEffectiveGDIScale())
	if resolved := r.resolvePrimaryFontPath(); resolved != "" {
		tr.SetFont(resolved)
	}
	r.textRenderer = tr
	return tr
}

func (r *Renderer) ensureDWriteRenderer() *DWriteRenderer {
	if r.dwriteRenderer != nil {
		return r.dwriteRenderer
	}
	dwr := NewDWriteRenderer("candidate")
	dwr.SetGDIParams(r.fontConfig.GetEffectiveGDIWeight(), r.fontConfig.GetEffectiveGDIScale())
	if resolved := r.resolvePrimaryFontPath(); resolved != "" {
		dwr.SetFont(resolved)
	}
	r.dwriteRenderer = dwr
	return dwr
}

// ensureFontCache prepares the gg/text font cache.
// 这里故意不复用 resolvePrimaryFontPath()，因为 gg/text 不能直接消费 TTC。
func (r *Renderer) ensureFontCache() *fontCache {
	if r.fontCache == nil {
		r.fontCache = newFontCache()
	}
	if r.fontPath != "" {
		r.fontConfig.SetPrimaryFont(r.fontPath)
	}
	// The gg/text path must resolve through the TTF-only chain; GDI / DirectWrite
	// keep using resolvePrimaryFontPath() so TTC system fonts still work there.
	resolved := r.fontConfig.ResolveTextPrimaryFont()
	if resolved == "" {
		return r.fontCache
	}
	r.fontCache.mu.Lock()
	defer r.fontCache.mu.Unlock()
	_ = r.fontCache.loadFont(resolved)
	r.fontReady = true
	return r.fontCache
}

func (r *Renderer) releaseGDIBackend() {
	if r.textRenderer != nil {
		r.textRenderer.Close()
		r.textRenderer = nil
	}
}

func (r *Renderer) releaseDWriteBackend() {
	if r.dwriteRenderer != nil {
		r.dwriteRenderer.Close()
		r.dwriteRenderer = nil
	}
}

func (r *Renderer) releaseFreeTypeBackend() {
	if r.fontCache != nil {
		r.fontCache.Close()
		r.fontCache = nil
	}
	r.fontReady = false
}

// updateTextDrawer recreates the active backend and releases the inactive ones.
// 这样可以避免不同文字后端长期同时持有字体或设备资源。
func (r *Renderer) updateTextDrawer() {
	switch r.config.TextRenderMode {
	case TextRenderModeFreetype:
		fc := r.ensureFontCache()
		r.releaseGDIBackend()
		r.releaseDWriteBackend()
		r.textDrawer = newFreeTypeDrawer(fc, r.fontConfig)
	case TextRenderModeDirectWrite:
		dwr := r.ensureDWriteRenderer()
		if dwr != nil && dwr.IsAvailable() {
			r.releaseGDIBackend()
			r.releaseFreeTypeBackend()
			r.textDrawer = newDirectWriteDrawer(dwr)
			return
		}
		r.releaseDWriteBackend()
		tr := r.ensureTextRenderer()
		r.releaseFreeTypeBackend()
		r.textDrawer = newGDIDrawer(tr)
	default:
		tr := r.ensureTextRenderer()
		r.releaseDWriteBackend()
		r.releaseFreeTypeBackend()
		r.textDrawer = newGDIDrawer(tr)
	}
}

// SetGDIFontParams updates GDI font weight and scale for text rendering
func (r *Renderer) SetGDIFontParams(weight int, scale float64) {
	r.fontConfig.SetGDIFontWeight(weight)
	r.fontConfig.SetGDIFontScale(scale)
	if r.textRenderer != nil {
		r.textRenderer.SetGDIParams(weight, scale)
	}
	if r.dwriteRenderer != nil {
		r.dwriteRenderer.SetGDIParams(weight, scale)
	}
}

// SetTextRenderMode switches between GDI, gg/text, and DirectWrite rendering.
func (r *Renderer) SetTextRenderMode(mode TextRenderMode) {
	r.config.TextRenderMode = mode
	r.updateTextDrawer()
}

// GetTextRenderMode returns the current text rendering mode
func (r *Renderer) GetTextRenderMode() TextRenderMode {
	switch r.config.TextRenderMode {
	case TextRenderModeFreetype:
		return TextRenderModeFreetype
	case TextRenderModeDirectWrite:
		return TextRenderModeDirectWrite
	default:
		return TextRenderModeGDI
	}
}

// SetFontPath sets the font path
func (r *Renderer) SetFontPath(path string) {
	r.fontPath = path
	r.fontReady = false
	resolved := r.resolvePrimaryFontPath()
	textResolved := r.fontConfig.ResolveTextPrimaryFont()
	if r.fontCache != nil && resolved != "" {
		r.fontCache.mu.Lock()
		// gg/text 使用经过 TTF-only 过滤的路径；GDI / DWrite 仍用 resolved。
		_ = r.fontCache.loadFont(textResolved)
		r.fontCache.mu.Unlock()
		r.fontReady = textResolved != ""
	}
	if r.textRenderer != nil && resolved != "" {
		r.textRenderer.SetFont(resolved)
	}
	if r.dwriteRenderer != nil && resolved != "" {
		r.dwriteRenderer.SetFont(resolved)
	}
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
		resolved := r.resolvePrimaryFontPath()
		textResolved := r.fontConfig.ResolveTextPrimaryFont()
		if r.fontCache != nil && resolved != "" {
			r.fontCache.mu.Lock()
			_ = r.fontCache.loadFont(textResolved)
			r.fontCache.mu.Unlock()
			r.fontReady = textResolved != ""
		}
		if r.textRenderer != nil && resolved != "" {
			r.textRenderer.SetFont(resolved)
		}
		if r.dwriteRenderer != nil && resolved != "" {
			r.dwriteRenderer.SetFont(resolved)
		}
	}
}

// Close releases renderer-owned font and text resources.
func (r *Renderer) Close() {
	r.releaseFreeTypeBackend()
	r.releaseGDIBackend()
	r.releaseDWriteBackend()
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

func radians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func (r *Renderer) drawRoundedRect(dc *gg.Context, x, y, w, h, radius float64) {
	dc.NewSubPath()
	dc.MoveTo(x+radius, y)
	dc.LineTo(x+w-radius, y)
	dc.DrawArc(x+w-radius, y+radius, radius, -radians(90), 0)
	dc.LineTo(x+w, y+h-radius)
	dc.DrawArc(x+w-radius, y+h-radius, radius, 0, radians(90))
	dc.LineTo(x+radius, y+h)
	dc.DrawArc(x+radius, y+h-radius, radius, radians(90), radians(180))
	dc.LineTo(x, y+radius)
	dc.DrawArc(x+radius, y+radius, radius, radians(180), radians(270))
	dc.ClosePath()
}

// RenderModeIndicator renders a mode indicator with adaptive width
func (r *Renderer) RenderModeIndicator(mode string) *image.RGBA {
	scale := GetDPIScale()
	td := r.textDrawer

	minWidth := 50.0 * scale
	height := 36.0 * scale
	fontSize := 20.0 * scale
	padding := 12.0 * scale

	// Measure text width
	textWidth := td.MeasureString(mode, fontSize)

	// Adaptive width: max(minWidth, textWidth + padding*2)
	width := textWidth + padding*2
	if width < minWidth {
		width = minWidth
	}

	dc := gg.NewContext(int(width), int(height))

	// Get colors from theme
	bgColor, textColor := r.getModeIndicatorColors()

	// Draw background shape
	dc.SetColor(bgColor)
	r.drawRoundedRect(dc, 2*scale, 2*scale, width-4*scale, height-4*scale, 6*scale)
	dc.Fill()

	// Draw mode text
	img := dc.Image().(*image.RGBA)
	td.BeginDraw(img)
	tw := td.MeasureString(mode, fontSize)
	td.DrawString(mode, width/2-tw/2, height/2+7*scale, fontSize, textColor)
	td.EndDraw()

	return img
}
