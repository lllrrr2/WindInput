package ui

// TextBackendManager manages the lifecycle of GDI / FreeType(gg/text) / DirectWrite
// text rendering backends. Embed this in any struct that needs text rendering.
// Callers are responsible for their own locking if needed.
type TextBackendManager struct {
	fontCache      *fontCache
	textRenderer   *TextRenderer
	dwriteRenderer *DWriteRenderer
	textDrawer     TextDrawer
	fontConfig     *FontConfig
	fontPath       string
	fontReady      bool
	label          string // identifier for DWriteRenderer (e.g., "candidate", "toolbar")
}

// NewTextBackendManager creates a TextBackendManager with the given label and default font config.
func NewTextBackendManager(label string) TextBackendManager {
	return TextBackendManager{
		fontConfig: NewFontConfig(),
		label:      label,
	}
}

// FontConfig returns the underlying FontConfig.
func (m *TextBackendManager) FontConfig() *FontConfig {
	return m.fontConfig
}

// TextDrawer returns the currently active TextDrawer.
func (m *TextBackendManager) TextDrawer() TextDrawer {
	return m.textDrawer
}

// FontPath returns the current font path.
func (m *TextBackendManager) FontPath() string {
	return m.fontPath
}

// FontReady returns whether a font has been successfully loaded.
func (m *TextBackendManager) FontReady() bool {
	return m.fontReady
}

// ResolvePrimaryFontPath resolves and caches the primary font path.
// GDI / DirectWrite allow TTC; this returns the general resolved path.
func (m *TextBackendManager) ResolvePrimaryFontPath() string {
	if m.fontPath != "" {
		m.fontConfig.SetPrimaryFont(m.fontPath)
	}
	resolved := m.fontConfig.ResolvePrimaryFont()
	if resolved != "" {
		m.fontPath = resolved
	}
	return resolved
}

// EnsureTextRenderer lazily creates the GDI TextRenderer.
func (m *TextBackendManager) EnsureTextRenderer() *TextRenderer {
	if m.textRenderer != nil {
		return m.textRenderer
	}
	tr := NewTextRenderer()
	tr.SetGDIParams(m.fontConfig.GetEffectiveGDIWeight(), m.fontConfig.GetEffectiveGDIScale())
	if resolved := m.ResolvePrimaryFontPath(); resolved != "" {
		tr.SetFont(resolved)
	}
	m.textRenderer = tr
	return tr
}

// EnsureDWriteRenderer lazily creates the DirectWrite renderer.
func (m *TextBackendManager) EnsureDWriteRenderer() *DWriteRenderer {
	if m.dwriteRenderer != nil {
		return m.dwriteRenderer
	}
	dwr := NewDWriteRenderer(m.label)
	dwr.SetGDIParams(m.fontConfig.GetEffectiveGDIWeight(), m.fontConfig.GetEffectiveGDIScale())
	if resolved := m.ResolvePrimaryFontPath(); resolved != "" {
		dwr.SetFont(resolved)
	}
	m.dwriteRenderer = dwr
	return dwr
}

// EnsureFontCache lazily creates the gg/text font cache.
// Uses TTF-only resolution since gg/text cannot handle TTC.
func (m *TextBackendManager) EnsureFontCache() *fontCache {
	if m.fontCache == nil {
		m.fontCache = newFontCache()
	}
	if m.fontPath != "" {
		m.fontConfig.SetPrimaryFont(m.fontPath)
	}
	resolved := m.fontConfig.ResolveTextPrimaryFont()
	if resolved == "" {
		return m.fontCache
	}
	m.fontCache.mu.Lock()
	_ = m.fontCache.loadFont(resolved)
	m.fontCache.mu.Unlock()
	m.fontReady = true
	return m.fontCache
}

// ReleaseGDIBackend closes and clears the GDI TextRenderer.
func (m *TextBackendManager) ReleaseGDIBackend() {
	if m.textRenderer != nil {
		m.textRenderer.Close()
		m.textRenderer = nil
	}
}

// ReleaseDWriteBackend closes and clears the DirectWrite renderer.
func (m *TextBackendManager) ReleaseDWriteBackend() {
	if m.dwriteRenderer != nil {
		m.dwriteRenderer.Close()
		m.dwriteRenderer = nil
	}
}

// ReleaseFreeTypeBackend closes and clears the gg/text font cache.
func (m *TextBackendManager) ReleaseFreeTypeBackend() {
	if m.fontCache != nil {
		m.fontCache.Close()
		m.fontCache = nil
	}
	m.fontReady = false
}

// SetTextRenderMode switches between GDI, FreeType(gg/text), and DirectWrite.
// Releases inactive backends to avoid holding resources from multiple backends.
func (m *TextBackendManager) SetTextRenderMode(mode TextRenderMode) {
	switch mode {
	case TextRenderModeFreetype:
		fc := m.EnsureFontCache()
		m.ReleaseGDIBackend()
		m.ReleaseDWriteBackend()
		m.textDrawer = newFreeTypeDrawer(fc, m.fontConfig)
	case TextRenderModeDirectWrite:
		dwr := m.EnsureDWriteRenderer()
		if dwr != nil && dwr.IsAvailable() {
			m.ReleaseGDIBackend()
			m.ReleaseFreeTypeBackend()
			m.textDrawer = newDirectWriteDrawer(dwr)
			return
		}
		m.ReleaseDWriteBackend()
		tr := m.EnsureTextRenderer()
		m.ReleaseFreeTypeBackend()
		m.textDrawer = newGDIDrawer(tr)
	default:
		tr := m.EnsureTextRenderer()
		m.ReleaseDWriteBackend()
		m.ReleaseFreeTypeBackend()
		m.textDrawer = newGDIDrawer(tr)
	}
}

// SetGDIFontParams updates GDI font weight and scale on active renderers.
func (m *TextBackendManager) SetGDIFontParams(weight int, scale float64) {
	m.fontConfig.SetGDIFontWeight(weight)
	m.fontConfig.SetGDIFontScale(scale)
	if m.textRenderer != nil {
		m.textRenderer.SetGDIParams(weight, scale)
	}
	if m.dwriteRenderer != nil {
		m.dwriteRenderer.SetGDIParams(weight, scale)
	}
}

// SetFontPath updates the primary font on all active backends.
func (m *TextBackendManager) SetFontPath(path string) {
	m.fontPath = path
	resolved := m.ResolvePrimaryFontPath()
	textResolved := m.fontConfig.ResolveTextPrimaryFont()
	if m.fontCache != nil && textResolved != "" {
		m.fontCache.mu.Lock()
		_ = m.fontCache.loadFont(textResolved)
		m.fontCache.mu.Unlock()
		m.fontReady = true
	}
	if m.textRenderer != nil && resolved != "" {
		m.textRenderer.SetFont(resolved)
	}
	if m.dwriteRenderer != nil && resolved != "" {
		m.dwriteRenderer.SetFont(resolved)
	}
}

// Close releases all backends.
func (m *TextBackendManager) Close() {
	m.ReleaseFreeTypeBackend()
	m.ReleaseGDIBackend()
	m.ReleaseDWriteBackend()
}
