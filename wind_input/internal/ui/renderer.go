package ui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"

	"github.com/golang/freetype/truetype"
	"github.com/fogleman/gg"
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
	config    RenderConfig
	fontPath  string
	fontCache *fontCache
	fontReady bool
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

// RenderCandidates renders candidates to an image
// Optimized to minimize font loading operations
func (r *Renderer) RenderCandidates(candidates []Candidate, input string, page, totalPages int) *image.RGBA {
	cfg := r.config
	scale := GetDPIScale()

	// Calculate dimensions with DPI scaling
	candidateCount := len(candidates)
	if candidateCount == 0 {
		candidateCount = 1 // Show at least input area
	}

	width := 280.0 * scale
	inputHeight := 30.0 * scale
	contentHeight := float64(candidateCount) * cfg.ItemHeight
	pageInfoHeight := 0.0
	if totalPages > 1 {
		pageInfoHeight = 24.0 * scale
	}
	height := cfg.Padding*2 + inputHeight + contentHeight + pageInfoHeight + 4*scale

	// Create context
	dc := gg.NewContext(int(width), int(height))

	// Draw shadow (simplified - just 1 layer instead of 4)
	dc.SetColor(color.RGBA{0, 0, 0, 15})
	r.drawRoundedRect(dc, 2, 2, width, height, cfg.CornerRadius)
	dc.Fill()

	// Draw background
	dc.SetColor(cfg.BackgroundColor)
	r.drawRoundedRect(dc, 0, 0, width-2, height-2, cfg.CornerRadius)
	dc.Fill()

	// Draw border
	dc.SetColor(cfg.BorderColor)
	dc.SetLineWidth(1)
	r.drawRoundedRect(dc, 0.5, 0.5, width-3, height-3, cfg.CornerRadius)
	dc.Stroke()

	// Get cached font faces
	mainFace := r.fontCache.getFace(cfg.FontSize)
	smallFace := r.fontCache.getFace(cfg.IndexFontSize)
	pageFace := r.fontCache.getFace(12 * scale)

	y := cfg.Padding

	// Draw input area
	dc.SetColor(cfg.InputBgColor)
	r.drawRoundedRect(dc, cfg.Padding, y, width-cfg.Padding*2-2, inputHeight, 4*scale)
	dc.Fill()

	// Draw input text
	if mainFace != nil {
		dc.SetFontFace(mainFace)
		dc.SetColor(cfg.InputTextColor)
		dc.DrawString(input, cfg.Padding+8*scale, y+inputHeight/2+cfg.FontSize/3)
	}
	y += inputHeight + 4*scale

	// First pass: draw all index circles (no font needed)
	for i := range candidates {
		itemY := y + float64(i)*cfg.ItemHeight
		indexX := cfg.Padding + 14*scale
		indexY := itemY + cfg.ItemHeight/2
		dc.SetColor(cfg.IndexBgColor)
		dc.DrawCircle(indexX, indexY, 11*scale)
		dc.Fill()
	}

	// Second pass: draw all index numbers (small font)
	if smallFace != nil {
		dc.SetFontFace(smallFace)
		dc.SetColor(cfg.IndexColor)
		for i, cand := range candidates {
			itemY := y + float64(i)*cfg.ItemHeight
			indexX := cfg.Padding + 14*scale
			indexY := itemY + cfg.ItemHeight/2
			indexStr := string(rune('0' + cand.Index))
			tw, _ := dc.MeasureString(indexStr)
			dc.DrawString(indexStr, indexX-tw/2, indexY+cfg.IndexFontSize/3)
		}
	}

	// Third pass: draw all candidate texts (main font)
	type commentInfo struct {
		text string
		x    float64
		y    float64
	}
	var comments []commentInfo

	if mainFace != nil {
		dc.SetFontFace(mainFace)
		dc.SetColor(cfg.TextColor)
		for i, cand := range candidates {
			itemY := y + float64(i)*cfg.ItemHeight
			dc.DrawString(cand.Text, cfg.Padding+32*scale, itemY+cfg.ItemHeight/2+cfg.FontSize/3)

			if cand.Comment != "" {
				candWidth, _ := dc.MeasureString(cand.Text)
				comments = append(comments, commentInfo{
					text: cand.Comment,
					x:    cfg.Padding + 32*scale + candWidth + 8*scale,
					y:    itemY + cfg.ItemHeight/2 + cfg.IndexFontSize/3,
				})
			}
		}
	}

	// Fourth pass: draw all comments (small font)
	if len(comments) > 0 && smallFace != nil {
		dc.SetFontFace(smallFace)
		dc.SetColor(color.RGBA{150, 150, 150, 255})
		for _, c := range comments {
			dc.DrawString(c.text, c.x, c.y)
		}
	}

	// Draw page info
	if totalPages > 1 && pageFace != nil {
		pageY := y + float64(len(candidates))*cfg.ItemHeight + 4*scale
		dc.SetFontFace(pageFace)
		dc.SetColor(cfg.InputTextColor)
		pageText := fmt.Sprintf("%d / %d  (← →)", page, totalPages)
		tw, _ := dc.MeasureString(pageText)
		dc.DrawString(pageText, width/2-tw/2, pageY+16*scale)
	}

	return dc.Image().(*image.RGBA)
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

// RenderModeIndicator renders a mode indicator (中/En)
func (r *Renderer) RenderModeIndicator(mode string) *image.RGBA {
	scale := GetDPIScale()

	width := 50.0 * scale
	height := 36.0 * scale
	fontSize := 20.0 * scale

	dc := gg.NewContext(int(width), int(height))

	// Draw background
	dc.SetColor(color.RGBA{50, 50, 50, 230})
	r.drawRoundedRect(dc, 2*scale, 2*scale, width-4*scale, height-4*scale, 6*scale)
	dc.Fill()

	// Use cached font face
	face := r.fontCache.getFace(fontSize)
	if face != nil {
		dc.SetFontFace(face)
	}

	// Draw mode text
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	tw, _ := dc.MeasureString(mode)
	dc.DrawString(mode, width/2-tw/2, height/2+7*scale)

	return dc.Image().(*image.RGBA)
}
