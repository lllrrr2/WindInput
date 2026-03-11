package ui

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
)

// TextRenderMode defines the text rendering backend
type TextRenderMode string

const (
	TextRenderModeGDI      TextRenderMode = "gdi"      // Windows GDI native rendering
	TextRenderModeFreetype TextRenderMode = "freetype" // FreeType rendering (original)
)

// TextDrawer provides a unified interface for text measurement and drawing.
// Both FreeType and GDI backends implement this interface.
// The interface is designed to be engine-agnostic so that rendering backends
// (FreeType, GDI, DirectWrite, etc.) can be swapped transparently.
type TextDrawer interface {
	// SetFont sets the font by file path. The backend resolves it to the
	// appropriate internal representation (e.g., GDI family name, freetype font).
	SetFont(fontPath string)
	// MeasureString measures text width in pixels for the given font size.
	MeasureString(text string, fontSize float64) float64
	// BeginDraw prepares for text drawing on the given image.
	// Must be called before DrawString, and EndDraw must be called after.
	BeginDraw(img *image.RGBA)
	// DrawString draws text at baseline position (x, y), matching gg.DrawString coordinates.
	DrawString(text string, x, y float64, fontSize float64, clr color.Color)
	// EndDraw finalizes text drawing and flushes results to the image.
	EndDraw()
	// Close releases all resources held by this drawer.
	Close()
}

// --- FreeType (original) implementation ---

// freeTypeDrawer wraps gg/freetype for text rendering, preserving the original rendering behavior.
type freeTypeDrawer struct {
	cache *fontCache
	dc    *gg.Context
}

func newFreeTypeDrawer(cache *fontCache) *freeTypeDrawer {
	return &freeTypeDrawer{cache: cache}
}

func (d *freeTypeDrawer) SetFont(fontPath string) {
	d.cache.mu.Lock()
	defer d.cache.mu.Unlock()
	d.cache.loadFont(fontPath)
}

func (d *freeTypeDrawer) MeasureString(text string, fontSize float64) float64 {
	if text == "" {
		return 0
	}
	face := d.cache.getFace(fontSize)
	if face == nil {
		return 0
	}
	dc := d.dc
	if dc == nil {
		dc = gg.NewContext(1, 1)
	}
	dc.SetFontFace(face)
	w, _ := dc.MeasureString(text)
	return w
}

func (d *freeTypeDrawer) BeginDraw(img *image.RGBA) {
	d.dc = gg.NewContextForRGBA(img)
}

func (d *freeTypeDrawer) DrawString(text string, x, y float64, fontSize float64, clr color.Color) {
	if d.dc == nil || text == "" {
		return
	}
	face := d.cache.getFace(fontSize)
	if face == nil {
		return
	}
	d.dc.SetFontFace(face)
	d.dc.SetColor(clr)
	d.dc.DrawString(text, x, y)
}

func (d *freeTypeDrawer) EndDraw() {
	d.dc = nil
}

func (d *freeTypeDrawer) Close() {
	// fontCache is shared, not owned by this drawer
	d.dc = nil
}

// --- GDI implementation ---

// gdiDrawer wraps TextRenderer for Windows-native GDI text rendering.
type gdiDrawer struct {
	tr *TextRenderer
}

func newGDIDrawer(tr *TextRenderer) *gdiDrawer {
	return &gdiDrawer{tr: tr}
}

func (d *gdiDrawer) SetFont(fontPath string) {
	d.tr.SetFont(fontPath)
}

func (d *gdiDrawer) MeasureString(text string, fontSize float64) float64 {
	return d.tr.MeasureString(text, fontSize)
}

func (d *gdiDrawer) BeginDraw(img *image.RGBA) {
	d.tr.BeginDraw(img)
}

func (d *gdiDrawer) DrawString(text string, x, y float64, fontSize float64, clr color.Color) {
	d.tr.DrawString(text, x, y, fontSize, clr)
}

func (d *gdiDrawer) EndDraw() {
	d.tr.EndDraw()
}

func (d *gdiDrawer) Close() {
	d.tr.Close()
}
