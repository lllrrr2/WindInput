package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/fogleman/gg"
)

// RenderConfig contains rendering configuration
type RenderConfig struct {
	FontPath       string
	FontSize       float64
	IndexFontSize  float64
	Padding        float64
	ItemHeight     float64
	CornerRadius   float64
	BackgroundColor color.Color
	TextColor       color.Color
	IndexColor      color.Color
	IndexBgColor    color.Color
	InputBgColor    color.Color
	InputTextColor  color.Color
	BorderColor     color.Color
}

// DefaultRenderConfig returns default rendering configuration
func DefaultRenderConfig() RenderConfig {
	return RenderConfig{
		FontPath:        "",  // Will use system font
		FontSize:        18,
		IndexFontSize:   14,
		Padding:         10,
		ItemHeight:      32,
		CornerRadius:    8,
		BackgroundColor: color.RGBA{255, 255, 255, 245}, // Slightly transparent white
		TextColor:       color.RGBA{30, 30, 30, 255},
		IndexColor:      color.RGBA{255, 255, 255, 255},
		IndexBgColor:    color.RGBA{66, 133, 244, 255}, // Blue
		InputBgColor:    color.RGBA{240, 240, 240, 255},
		InputTextColor:  color.RGBA{100, 100, 100, 255},
		BorderColor:     color.RGBA{200, 200, 200, 255},
	}
}

// Renderer renders candidate window content
type Renderer struct {
	config   RenderConfig
	fontPath string
}

// NewRenderer creates a new renderer
func NewRenderer(config RenderConfig) *Renderer {
	return &Renderer{
		config: config,
	}
}

// SetFontPath sets the font path
func (r *Renderer) SetFontPath(path string) {
	r.fontPath = path
}

// RenderCandidates renders candidates to an image
func (r *Renderer) RenderCandidates(candidates []Candidate, input string, page, totalPages int) *image.RGBA {
	cfg := r.config

	// Calculate dimensions
	candidateCount := len(candidates)
	if candidateCount == 0 {
		candidateCount = 1 // Show at least input area
	}

	width := 280.0
	inputHeight := 30.0
	contentHeight := float64(candidateCount) * cfg.ItemHeight
	pageInfoHeight := 0.0
	if totalPages > 1 {
		pageInfoHeight = 24.0
	}
	height := cfg.Padding*2 + inputHeight + contentHeight + pageInfoHeight + 4 // 4 for gaps

	// Create context
	dc := gg.NewContext(int(width), int(height))

	// Draw rounded rectangle background with shadow effect
	r.drawRoundedRectWithShadow(dc, 0, 0, width, height, cfg.CornerRadius)

	// Draw background
	dc.SetColor(cfg.BackgroundColor)
	r.drawRoundedRect(dc, 2, 2, width-4, height-4, cfg.CornerRadius-1)
	dc.Fill()

	// Draw border
	dc.SetColor(cfg.BorderColor)
	dc.SetLineWidth(1)
	r.drawRoundedRect(dc, 1, 1, width-2, height-2, cfg.CornerRadius)
	dc.Stroke()

	// Load font
	fontLoaded := false
	if r.fontPath != "" {
		if err := dc.LoadFontFace(r.fontPath, cfg.FontSize); err == nil {
			fontLoaded = true
		}
	}
	if !fontLoaded {
		// Try common Windows fonts
		fonts := []string{
			"C:/Windows/Fonts/msyh.ttc",    // Microsoft YaHei
			"C:/Windows/Fonts/simhei.ttf",  // SimHei
			"C:/Windows/Fonts/simsun.ttc",  // SimSun
			"C:/Windows/Fonts/arial.ttf",   // Arial fallback
		}
		for _, f := range fonts {
			if err := dc.LoadFontFace(f, cfg.FontSize); err == nil {
				fontLoaded = true
				r.fontPath = f
				break
			}
		}
	}

	y := cfg.Padding

	// Draw input area
	dc.SetColor(cfg.InputBgColor)
	r.drawRoundedRect(dc, cfg.Padding, y, width-cfg.Padding*2, inputHeight, 4)
	dc.Fill()

	dc.SetColor(cfg.InputTextColor)
	if fontLoaded {
		dc.DrawString(input, cfg.Padding+8, y+inputHeight/2+cfg.FontSize/3)
	}
	y += inputHeight + 4

	// Draw candidates
	for i, cand := range candidates {
		itemY := y + float64(i)*cfg.ItemHeight

		// Draw index circle
		indexX := cfg.Padding + 14
		indexY := itemY + cfg.ItemHeight/2
		dc.SetColor(cfg.IndexBgColor)
		dc.DrawCircle(indexX, indexY, 11)
		dc.Fill()

		// Draw index number
		dc.SetColor(cfg.IndexColor)
		if fontLoaded {
			dc.LoadFontFace(r.fontPath, cfg.IndexFontSize)
		}
		indexStr := string(rune('0' + cand.Index))
		tw, _ := dc.MeasureString(indexStr)
		dc.DrawString(indexStr, indexX-tw/2, indexY+cfg.IndexFontSize/3)

		// Draw candidate text
		dc.SetColor(cfg.TextColor)
		if fontLoaded {
			dc.LoadFontFace(r.fontPath, cfg.FontSize)
		}
		dc.DrawString(cand.Text, cfg.Padding+32, itemY+cfg.ItemHeight/2+cfg.FontSize/3)
	}

	// Draw page info
	if totalPages > 1 {
		pageY := y + float64(len(candidates))*cfg.ItemHeight + 4
		dc.SetColor(cfg.InputTextColor)
		if fontLoaded {
			dc.LoadFontFace(r.fontPath, 12)
		}
		pageText := fmt.Sprintf("%d / %d  (← →)", page, totalPages)
		tw, _ := dc.MeasureString(pageText)
		dc.DrawString(pageText, width/2-tw/2, pageY+16)
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

func (r *Renderer) drawRoundedRectWithShadow(dc *gg.Context, x, y, w, h, radius float64) {
	// Draw shadow layers
	for i := 3; i >= 0; i-- {
		offset := float64(i)
		alpha := uint8(20 - i*5)
		dc.SetColor(color.RGBA{0, 0, 0, alpha})
		r.drawRoundedRect(dc, x+offset, y+offset, w, h, radius)
		dc.Fill()
	}
}

// RenderModeIndicator renders a mode indicator (中/En)
func (r *Renderer) RenderModeIndicator(mode string) *image.RGBA {
	width := 50.0
	height := 36.0

	dc := gg.NewContext(int(width), int(height))

	// Draw background
	dc.SetColor(color.RGBA{50, 50, 50, 230})
	r.drawRoundedRect(dc, 2, 2, width-4, height-4, 6)
	dc.Fill()

	// Load font
	fontLoaded := false
	if r.fontPath != "" {
		if err := dc.LoadFontFace(r.fontPath, 20); err == nil {
			fontLoaded = true
		}
	}
	if !fontLoaded {
		fonts := []string{
			"C:/Windows/Fonts/msyh.ttc",
			"C:/Windows/Fonts/simhei.ttf",
			"C:/Windows/Fonts/arial.ttf",
		}
		for _, f := range fonts {
			if err := dc.LoadFontFace(f, 20); err == nil {
				r.fontPath = f
				break
			}
		}
	}

	// Draw mode text
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	tw, _ := dc.MeasureString(mode)
	dc.DrawString(mode, width/2-tw/2, height/2+7)

	return dc.Image().(*image.RGBA)
}
