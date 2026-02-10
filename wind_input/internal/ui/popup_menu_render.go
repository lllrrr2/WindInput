package ui

import (
	"image"
	"unsafe"

	"github.com/fogleman/gg"
)

// loadFont loads font for the context (Chinese text)
func (m *PopupMenu) loadFont(dc *gg.Context, fontSize float64) {
	fonts := []string{
		"C:/Windows/Fonts/msyh.ttc",
		"C:/Windows/Fonts/simhei.ttf",
		"C:/Windows/Fonts/simsun.ttc",
		"C:/Windows/Fonts/segoeui.ttf",
	}
	for _, path := range fonts {
		if err := dc.LoadFontFace(path, fontSize); err == nil {
			return
		}
	}
}

// loadSymbolFont loads a symbol-capable font for rendering ✓ ▸ etc.
func (m *PopupMenu) loadSymbolFont(dc *gg.Context, fontSize float64) {
	fonts := []string{
		"C:/Windows/Fonts/seguisym.ttf", // Segoe UI Symbol (Win7+, best coverage)
		"C:/Windows/Fonts/segmdl2.ttf",  // Segoe MDL2 Assets (Win10+)
		"C:/Windows/Fonts/segoeui.ttf",  // Segoe UI
		"C:/Windows/Fonts/arial.ttf",    // Arial
		"C:/Windows/Fonts/msyh.ttc",     // Microsoft YaHei (fallback)
	}
	for _, path := range fonts {
		if err := dc.LoadFontFace(path, fontSize); err == nil {
			return
		}
	}
}

// render renders the menu to an image
func (m *PopupMenu) render() *image.RGBA {
	m.mu.Lock()
	items := m.items
	hoverIdx := m.hoverIndex
	submenuIdx := m.submenuIndex
	width := m.width
	height := m.height
	hasChecked := m.hasChecked
	hasChildren := m.hasChildren
	colors := m.getPopupMenuColors()
	m.mu.Unlock()

	scale := GetDPIScale()
	fontSize := menuFontSize * scale
	itemH := int(float64(menuItemHeight) * scale)
	sepH := int(float64(menuSeparatorHeight) * scale)
	padX := float64(menuPaddingX) * scale
	padY := int(float64(menuPaddingY) * scale)
	checkW := 0.0
	if hasChecked {
		checkW = float64(menuCheckMarkWidth) * scale
	}
	arrowW := 0.0
	if hasChildren {
		arrowW = float64(menuArrowWidth) * scale
	}

	dc := gg.NewContext(width, height)
	m.loadFont(dc, fontSize)

	// Calculate corner radius with DPI scaling
	radius := float64(menuCornerRadius) * scale

	// Fill background with rounded rectangle
	dc.SetRGBA(1, 1, 1, 0) // Transparent background first
	dc.Clear()

	// Draw filled rounded rectangle for background
	dc.SetColor(colors.BackgroundColor)
	dc.DrawRoundedRectangle(0.5, 0.5, float64(width)-1, float64(height)-1, radius)
	dc.Fill()

	// Set clip to rounded rectangle so hover backgrounds don't overflow
	dc.DrawRoundedRectangle(1, 1, float64(width)-2, float64(height)-2, radius-1)
	dc.Clip()

	// Draw items
	y := padY
	for i, item := range items {
		if item.Separator {
			// Draw separator line
			sepY := float64(y + sepH/2)
			dc.SetColor(colors.SeparatorColor)
			dc.DrawLine(4*scale, sepY, float64(width)-4*scale, sepY)
			dc.Stroke()
			y += sepH
		} else {
			// Determine if this item should be highlighted
			isHovered := (i == hoverIdx && !item.Disabled) || (i == submenuIdx)

			// Draw item background if hovered or submenu is open for this item
			if isHovered {
				dc.SetColor(colors.HoverBgColor)
				dc.DrawRectangle(1, float64(y), float64(width-2), float64(itemH))
				dc.Fill()
			}

			// Draw check mark using symbol font
			if item.Checked {
				if item.Disabled {
					dc.SetColor(colors.DisabledColor)
				} else if isHovered {
					dc.SetColor(colors.HoverTextColor)
				} else {
					dc.SetColor(colors.TextColor)
				}
				m.loadSymbolFont(dc, fontSize)
				cx := padX/2 + checkW/2
				cy := float64(y) + float64(itemH)/2 + fontSize/3
				sw, _ := dc.MeasureString("✓")
				dc.DrawString("✓", cx-sw/2, cy)
				m.loadFont(dc, fontSize) // switch back to text font
			}

			// Draw text
			if item.Disabled {
				dc.SetColor(colors.DisabledColor)
			} else if isHovered {
				dc.SetColor(colors.HoverTextColor)
			} else {
				dc.SetColor(colors.TextColor)
			}

			textX := padX + checkW
			textY := float64(y) + float64(itemH)/2 + fontSize/3
			dc.DrawString(item.Text, textX, textY)

			// Draw submenu arrow using symbol font
			if len(item.Children) > 0 {
				if item.Disabled {
					dc.SetColor(colors.DisabledColor)
				} else if isHovered {
					dc.SetColor(colors.HoverTextColor)
				} else {
					dc.SetColor(colors.TextColor)
				}
				m.loadSymbolFont(dc, fontSize)
				ax := float64(width) - padX/2 - arrowW/2
				ay := float64(y) + float64(itemH)/2 + fontSize/3
				sw, _ := dc.MeasureString("▸")
				dc.DrawString("▸", ax-sw/2, ay)
				m.loadFont(dc, fontSize) // switch back to text font
			}

			y += itemH
		}
	}

	// Reset clip and draw border
	dc.ResetClip()
	dc.SetColor(colors.BorderColor)
	dc.DrawRoundedRectangle(0.5, 0.5, float64(width)-1, float64(height)-1, radius)
	dc.Stroke()

	return dc.Image().(*image.RGBA)
}

// updateWindow updates the layered window with the rendered image
func (m *PopupMenu) updateWindow() {
	img := m.render()

	m.mu.Lock()
	x, y := m.x, m.y
	width, height := m.width, m.height
	m.mu.Unlock()

	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return
	}
	defer procReleaseDC.Call(0, hdcScreen)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return
	}
	defer procDeleteDC.Call(hdcMem)

	bi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      -int32(height),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BI_RGB,
		},
	}

	var bits unsafe.Pointer
	hBitmap, _, _ := procCreateDIBSection.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&bi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&bits)),
		0, 0,
	)
	if hBitmap == 0 {
		return
	}
	defer procDeleteObject.Call(hBitmap)

	procSelectObject.Call(hdcMem, hBitmap)

	// Copy image data (RGBA to BGRA with premultiplied alpha)
	pixelCount := width * height
	dstSlice := unsafe.Slice((*byte)(bits), pixelCount*4)

	for i := 0; i < pixelCount; i++ {
		srcIdx := i * 4
		dstIdx := i * 4

		r := img.Pix[srcIdx+0]
		g := img.Pix[srcIdx+1]
		b := img.Pix[srcIdx+2]
		a := img.Pix[srcIdx+3]

		if a == 255 {
			dstSlice[dstIdx+0] = b
			dstSlice[dstIdx+1] = g
			dstSlice[dstIdx+2] = r
			dstSlice[dstIdx+3] = a
		} else if a == 0 {
			dstSlice[dstIdx+0] = 0
			dstSlice[dstIdx+1] = 0
			dstSlice[dstIdx+2] = 0
			dstSlice[dstIdx+3] = 0
		} else {
			dstSlice[dstIdx+0] = byte(uint16(b) * uint16(a) / 255)
			dstSlice[dstIdx+1] = byte(uint16(g) * uint16(a) / 255)
			dstSlice[dstIdx+2] = byte(uint16(r) * uint16(a) / 255)
			dstSlice[dstIdx+3] = a
		}
	}

	ptSrc := POINT{X: 0, Y: 0}
	ptDst := POINT{X: int32(x), Y: int32(y)}
	size := SIZE{Cx: int32(width), Cy: int32(height)}
	blend := BLENDFUNCTION{
		BlendOp:             AC_SRC_OVER,
		BlendFlags:          0,
		SourceConstantAlpha: 255,
		AlphaFormat:         AC_SRC_ALPHA,
	}

	procUpdateLayeredWindow.Call(
		uintptr(m.hwnd),
		hdcScreen,
		uintptr(unsafe.Pointer(&ptDst)),
		uintptr(unsafe.Pointer(&size)),
		hdcMem,
		uintptr(unsafe.Pointer(&ptSrc)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ULW_ALPHA,
	)
}

// trackMouseLeave enables mouse leave tracking
func (m *PopupMenu) trackMouseLeave() {
	tme := TRACKMOUSEEVENT{
		CbSize:    uint32(unsafe.Sizeof(TRACKMOUSEEVENT{})),
		DwFlags:   TME_LEAVE,
		HwndTrack: uintptr(m.hwnd),
	}
	procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
}
