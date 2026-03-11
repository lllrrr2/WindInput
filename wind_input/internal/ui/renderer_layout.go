package ui

import (
	"fmt"
	"image"

	"github.com/fogleman/gg"
)

// RenderCandidates renders candidates to an image
// Optimized to minimize font loading operations
// hoverIndex: index of the hovered candidate (-1 for none)
// Returns the rendered image and candidate bounding rectangles for hit testing
func (r *Renderer) RenderCandidates(candidates []Candidate, input string, cursorPos int, page, totalPages int, hoverIndex int, hoverPageBtn string) (*image.RGBA, *RenderResult) {
	cfg := r.config

	// Choose layout based on config
	if cfg.Layout == "horizontal" {
		return r.renderHorizontalCandidates(candidates, input, cursorPos, page, totalPages, hoverIndex, hoverPageBtn)
	}
	return r.renderVerticalCandidates(candidates, input, cursorPos, page, totalPages, hoverIndex, hoverPageBtn)
}

// renderVerticalCandidates renders candidates in vertical layout (traditional style)
func (r *Renderer) renderVerticalCandidates(candidates []Candidate, input string, cursorPos int, page, totalPages int, hoverIndex int, hoverPageBtn string) (*image.RGBA, *RenderResult) {
	cfg := r.config
	scale := GetDPIScale()

	// Calculate dimensions with DPI scaling
	candidateCount := len(candidates)
	if candidateCount == 0 {
		candidateCount = 1 // Show at least input area
	}

	width := 280.0 * scale

	// 动态调整宽度以适应长输入文本
	mainFace := r.fontCache.getFace(cfg.FontSize)
	if mainFace != nil && input != "" {
		tmpDc := gg.NewContext(1, 1)
		tmpDc.SetFontFace(mainFace)
		inputTextWidth, _ := tmpDc.MeasureString(input)
		minInputWidth := inputTextWidth + cfg.Padding*2 + 16*scale
		if minInputWidth > width {
			width = minInputWidth
		}
	}

	inputHeight := 30.0 * scale
	if cfg.HidePreedit {
		inputHeight = 0
	}
	contentHeight := float64(candidateCount) * cfg.ItemHeight
	pageInfoHeight := 0.0
	if totalPages > 1 {
		pageInfoHeight = 24.0 * scale
	}
	height := cfg.Padding*2 + inputHeight + contentHeight + pageInfoHeight + 4*scale
	if cfg.HidePreedit {
		height = cfg.Padding*2 + contentHeight + pageInfoHeight
	}

	// Create context
	dc := gg.NewContext(int(width), int(height))

	// Draw shadow (simplified - just 1 layer instead of 4)
	dc.SetColor(r.getShadowColor())
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

	// Get cached font faces (mainFace already obtained above for width calculation)
	if mainFace == nil {
		mainFace = r.fontCache.getFace(cfg.FontSize)
	}
	smallFace := r.fontCache.getFace(cfg.IndexFontSize)
	pageFace := r.fontCache.getFace(12 * scale)

	// For text-style index, use slightly larger fonts for index, comment, and page
	isTextIndex := cfg.IndexStyle == "text"
	indexTextFace := mainFace
	indexTextSize := cfg.FontSize
	commentFace := smallFace
	commentSize := cfg.IndexFontSize
	if isTextIndex {
		indexTextSize = cfg.FontSize + 2*scale
		indexTextFace = r.fontCache.getFace(indexTextSize)
		commentSize = cfg.IndexFontSize + 2*scale
		commentFace = r.fontCache.getFace(commentSize)
		pageFace = r.fontCache.getFace(14 * scale)
	}

	y := cfg.Padding

	// Draw input area (if not hidden)
	if !cfg.HidePreedit {
		dc.SetColor(cfg.InputBgColor)
		r.drawRoundedRect(dc, cfg.Padding, y, width-cfg.Padding*2-2, inputHeight, 4*scale)
		dc.Fill()

		// Draw input text and cursor
		if mainFace != nil {
			dc.SetFontFace(mainFace)
			textX := cfg.Padding + 8*scale
			textY := y + inputHeight/2 + cfg.FontSize/3

			dc.SetColor(cfg.InputTextColor)
			dc.DrawString(input, textX, textY)

			// Draw cursor indicator
			if cursorPos >= 0 && cursorPos <= len(input) {
				cursorText := input[:cursorPos]
				cursorX, _ := dc.MeasureString(cursorText)
				cursorDrawX := textX + cursorX
				cursorTopY := y + 4*scale
				cursorBottomY := y + inputHeight - 4*scale
				dc.SetColor(cfg.InputTextColor)
				dc.SetLineWidth(1.5 * scale)
				dc.DrawLine(cursorDrawX, cursorTopY, cursorDrawX, cursorBottomY)
				dc.Stroke()
			}
		}
		y += inputHeight + 4*scale
	}

	// Build candidate rectangles for hit testing
	result := &RenderResult{
		Rects: make([]CandidateRect, len(candidates)),
	}

	// Draw hover background first (if any)
	if hoverIndex >= 0 && hoverIndex < len(candidates) {
		itemY := y + float64(hoverIndex)*cfg.ItemHeight
		dc.SetColor(cfg.HoverBgColor)
		r.drawRoundedRect(dc, cfg.Padding-2, itemY, width-cfg.Padding*2+2, cfg.ItemHeight, 4*scale)
		dc.Fill()
	}

	// Draw accent bar (if enabled)
	if cfg.HasAccentBar && cfg.AccentBarColor != nil {
		barWidth := 3.0 * scale
		barMargin := height * 0.3
		if barMargin < cfg.CornerRadius+1 {
			barMargin = cfg.CornerRadius + 1
		}
		dc.SetColor(cfg.AccentBarColor)
		r.drawRoundedRect(dc, 1, barMargin, barWidth, height-barMargin*2, barWidth/2)
		dc.Fill()
	}

	// Determine text offset based on index style
	textStartX := cfg.Padding + 32*scale // default for circle style
	if cfg.IndexStyle == "text" {
		textStartX = cfg.Padding + 24*scale // less space needed for plain text index
	}

	// First pass: draw all index backgrounds (circles) and record bounding boxes
	for i := range candidates {
		itemY := y + float64(i)*cfg.ItemHeight

		if cfg.IndexStyle != "text" {
			// Circle style: draw filled circle
			indexX := cfg.Padding + 14*scale
			indexY := itemY + cfg.ItemHeight/2
			dc.SetColor(cfg.IndexBgColor)
			dc.DrawCircle(indexX, indexY, 11*scale)
			dc.Fill()
		}

		// Record bounding rectangle
		result.Rects[i] = CandidateRect{
			Index: i,
			X:     cfg.Padding - 2,
			Y:     itemY,
			W:     width - cfg.Padding*2 + 2,
			H:     cfg.ItemHeight,
		}
	}

	// Second pass: draw all index numbers
	if isTextIndex {
		// Text style: draw plain colored number (slightly larger font)
		if indexTextFace != nil {
			dc.SetFontFace(indexTextFace)
			dc.SetColor(cfg.IndexColor)
			for i, cand := range candidates {
				itemY := y + float64(i)*cfg.ItemHeight
				indexStr := string(rune('0' + cand.Index))
				dc.DrawString(indexStr, cfg.Padding+4*scale, itemY+cfg.ItemHeight/2+indexTextSize/3)
			}
		}
	} else {
		// Circle style: draw white number on circle
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
			dc.DrawString(cand.Text, textStartX, itemY+cfg.ItemHeight/2+cfg.FontSize/3)

			if cand.Comment != "" {
				candWidth, _ := dc.MeasureString(cand.Text)
				comments = append(comments, commentInfo{
					text: cand.Comment,
					x:    textStartX + candWidth + 8*scale,
					y:    itemY + cfg.ItemHeight/2 + commentSize/3,
				})
			}
		}
	}

	// Fourth pass: draw all comments (slightly larger font for text-style index)
	if len(comments) > 0 && commentFace != nil {
		dc.SetFontFace(commentFace)
		dc.SetColor(r.getCommentColor())
		for _, c := range comments {
			dc.DrawString(c.text, c.x, c.y)
		}
	}

	// Draw page info with clickable page buttons
	if totalPages > 1 && pageFace != nil {
		pageY := y + float64(len(candidates))*cfg.ItemHeight + 4*scale
		dc.SetFontFace(pageFace)

		pageText := fmt.Sprintf(" %d / %d ", page, totalPages)
		pageW, _ := dc.MeasureString(pageText)

		arrowSize := 8.0 * scale // Triangle size
		arrowPad := 8.0 * scale  // Padding around arrow
		arrowW := arrowSize + arrowPad*2
		totalW := arrowW + pageW + arrowW
		startX := width/2 - totalW/2
		centerY := pageY + 10*scale

		// Page up button rect
		pageUpBtnRect := CandidateRect{
			X: startX,
			Y: pageY,
			W: arrowW,
			H: 20 * scale,
		}

		// Draw page up hover background
		if hoverPageBtn == "up" {
			dc.SetColor(cfg.HoverBgColor)
			r.drawRoundedRect(dc, pageUpBtnRect.X, pageUpBtnRect.Y, pageUpBtnRect.W, pageUpBtnRect.H, 4*scale)
			dc.Fill()
		}

		// Draw left chevron (page up)
		leftArrowColor := cfg.IndexBgColor
		if page <= 1 {
			leftArrowColor = cfg.InputTextColor
		}
		leftCenterX := startX + arrowW/2
		dc.SetColor(leftArrowColor)
		r.drawChevronLeft(dc, leftCenterX, centerY, arrowSize, 1.5*scale)
		result.PageUpRect = &pageUpBtnRect

		// Draw page text
		dc.SetColor(cfg.InputTextColor)
		dc.DrawString(pageText, startX+arrowW, centerY+4*scale)

		// Page down button rect
		pageDownBtnRect := CandidateRect{
			X: startX + arrowW + pageW,
			Y: pageY,
			W: arrowW,
			H: 20 * scale,
		}

		// Draw page down hover background
		if hoverPageBtn == "down" {
			dc.SetColor(cfg.HoverBgColor)
			r.drawRoundedRect(dc, pageDownBtnRect.X, pageDownBtnRect.Y, pageDownBtnRect.W, pageDownBtnRect.H, 4*scale)
			dc.Fill()
		}

		// Draw right chevron (page down)
		rightArrowColor := cfg.IndexBgColor
		if page >= totalPages {
			rightArrowColor = cfg.InputTextColor
		}
		rightCenterX := startX + arrowW + pageW + arrowW/2
		dc.SetColor(rightArrowColor)
		r.drawChevronRight(dc, rightCenterX, centerY, arrowSize, 1.5*scale)
		result.PageDownRect = &pageDownBtnRect
	}

	return dc.Image().(*image.RGBA), result
}

// renderHorizontalCandidates renders candidates in horizontal layout (modern style)
func (r *Renderer) renderHorizontalCandidates(candidates []Candidate, input string, cursorPos int, page, totalPages int, hoverIndex int, hoverPageBtn string) (*image.RGBA, *RenderResult) {
	cfg := r.config
	scale := GetDPIScale()

	// Get cached font faces
	mainFace := r.fontCache.getFace(cfg.FontSize)
	smallFace := r.fontCache.getFace(cfg.IndexFontSize)
	pageFace := r.fontCache.getFace(12 * scale)

	// For text-style index, use slightly larger fonts for index, comment, and page
	isTextIndex := cfg.IndexStyle == "text"
	indexTextFace := mainFace
	indexTextSize := cfg.FontSize
	commentFace := smallFace
	commentSize := cfg.IndexFontSize
	if isTextIndex {
		indexTextSize = cfg.FontSize + 2*scale
		indexTextFace = r.fontCache.getFace(indexTextSize)
		commentSize = cfg.IndexFontSize + 2*scale
		commentFace = r.fontCache.getFace(commentSize)
		pageFace = r.fontCache.getFace(14 * scale)
	}

	// Measure all candidates to calculate total width
	type candMeasure struct {
		textWidth    float64
		commentWidth float64
		totalWidth   float64 // Total width including index, text, comment
	}
	measures := make([]candMeasure, len(candidates))

	indexSize := 18.0 * scale   // Index circle diameter
	indexMargin := 4.0 * scale  // Margin between index and text
	itemSpacing := 12.0 * scale // Space between items

	// For text-style index, adjust spacing
	if isTextIndex {
		indexMargin = 2.0 * scale  // Tighter margin for text-style (number is already spaced)
		itemSpacing = 16.0 * scale // More space between items for cleaner look
	}
	indexTextWidths := make([]float64, len(candidates))

	tmpDc := gg.NewContext(1, 1)

	// Measure index text widths for text style (use larger font)
	if isTextIndex && indexTextFace != nil {
		tmpDc.SetFontFace(indexTextFace)
		for i, cand := range candidates {
			indexStr := string(rune('0' + cand.Index))
			w, _ := tmpDc.MeasureString(indexStr)
			indexTextWidths[i] = w
		}
	}

	if mainFace != nil {
		tmpDc.SetFontFace(mainFace)
		for i, cand := range candidates {
			tw, _ := tmpDc.MeasureString(cand.Text)
			measures[i].textWidth = tw
		}
	}
	if commentFace != nil {
		tmpDc.SetFontFace(commentFace)
		for i, cand := range candidates {
			if cand.Comment != "" {
				cw, _ := tmpDc.MeasureString(cand.Comment)
				measures[i].commentWidth = cw
			}
		}
	}

	// Calculate total width for each candidate
	for i, cand := range candidates {
		if isTextIndex {
			measures[i].totalWidth = indexTextWidths[i] + indexMargin + measures[i].textWidth
		} else {
			measures[i].totalWidth = indexSize + indexMargin + measures[i].textWidth
		}
		if cand.Comment != "" {
			measures[i].totalWidth += 6*scale + measures[i].commentWidth
		}
	}

	// Calculate total candidates width
	candidatesWidth := 0.0
	for i := range candidates {
		candidatesWidth += measures[i].totalWidth
		if i < len(candidates)-1 {
			candidatesWidth += itemSpacing
		}
	}

	// Page info width (including arrow buttons)
	arrowSize := 8.0 * scale
	arrowPad := 6.0 * scale
	arrowW := arrowSize + arrowPad*2
	pageInfoWidth := 0.0
	if totalPages > 1 && pageFace != nil {
		tmpDc.SetFontFace(pageFace)
		pageText := fmt.Sprintf(" %d/%d ", page, totalPages)
		textW, _ := tmpDc.MeasureString(pageText)
		pageInfoWidth = arrowW + textW + arrowW + 8*scale // arrows + text + spacing
	}

	// Input area (preedit)
	inputWidth := 0.0
	inputHeight := 0.0
	if !cfg.HidePreedit && input != "" {
		if mainFace != nil {
			tmpDc.SetFontFace(mainFace)
			inputWidth, _ = tmpDc.MeasureString(input)
			inputWidth += 16 * scale // padding
		}
		inputHeight = 24 * scale
	}

	// Extra padding for accent bar
	accentBarExtra := 0.0
	if cfg.HasAccentBar && cfg.AccentBarColor != nil {
		accentBarExtra = 3.0*scale + 2*scale
	}

	// Total width
	minWidth := 200.0 * scale
	contentWidth := cfg.Padding*2 + accentBarExtra + candidatesWidth + pageInfoWidth
	if inputWidth > 0 {
		contentWidth = cfg.Padding*2 + accentBarExtra + inputWidth
		if accentBarExtra+candidatesWidth+pageInfoWidth > accentBarExtra+inputWidth {
			contentWidth = cfg.Padding*2 + accentBarExtra + candidatesWidth + pageInfoWidth
		}
	}
	width := contentWidth
	if width < minWidth {
		width = minWidth
	}

	// Height calculation
	candidateRowHeight := 32.0 * scale
	height := cfg.Padding*2 + candidateRowHeight
	if inputHeight > 0 {
		height += inputHeight + 4*scale
	}

	// Create context
	dc := gg.NewContext(int(width), int(height))

	// Draw shadow
	dc.SetColor(r.getShadowColor())
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

	y := cfg.Padding

	// Draw accent bar (if enabled) - draw before preedit so it's behind everything
	accentBarOffset := 0.0
	if cfg.HasAccentBar && cfg.AccentBarColor != nil {
		barWidth := 3.0 * scale
		barMargin := height * 0.3
		if barMargin < cfg.CornerRadius+1 {
			barMargin = cfg.CornerRadius + 1
		}
		dc.SetColor(cfg.AccentBarColor)
		r.drawRoundedRect(dc, 1, barMargin, barWidth, height-barMargin*2, barWidth/2)
		dc.Fill()
		accentBarOffset = barWidth + 2*scale
	}

	// Draw input area (preedit) - if not hidden
	if !cfg.HidePreedit && input != "" {
		preeditX := cfg.Padding + accentBarOffset
		dc.SetColor(cfg.InputBgColor)
		r.drawRoundedRect(dc, preeditX, y, width-preeditX-cfg.Padding-2, inputHeight, 4*scale)
		dc.Fill()

		if mainFace != nil {
			dc.SetFontFace(mainFace)
			textX := preeditX + 8*scale
			textY := y + inputHeight/2 + cfg.FontSize/3

			dc.SetColor(cfg.InputTextColor)
			dc.DrawString(input, textX, textY)

			// Draw cursor indicator
			if cursorPos >= 0 && cursorPos <= len(input) {
				cursorText := input[:cursorPos]
				cursorXOffset, _ := dc.MeasureString(cursorText)
				cursorDrawX := textX + cursorXOffset
				cursorTopY := y + 3*scale
				cursorBottomY := y + inputHeight - 3*scale
				dc.SetColor(cfg.InputTextColor)
				dc.SetLineWidth(1.5 * scale)
				dc.DrawLine(cursorDrawX, cursorTopY, cursorDrawX, cursorBottomY)
				dc.Stroke()
			}
		}
		y += inputHeight + 4*scale
	}

	// Build candidate rectangles for hit testing
	result := &RenderResult{
		Rects: make([]CandidateRect, len(candidates)),
	}

	// Draw candidates horizontally
	x := cfg.Padding + accentBarOffset
	candY := y + candidateRowHeight/2

	for i, cand := range candidates {
		itemWidth := measures[i].totalWidth

		// Record bounding rectangle (before drawing)
		result.Rects[i] = CandidateRect{
			Index: i,
			X:     x - 4,
			Y:     y,
			W:     itemWidth + 8,
			H:     candidateRowHeight,
		}

		// Draw hover background if this is the hovered item
		if i == hoverIndex {
			dc.SetColor(cfg.HoverBgColor)
			r.drawRoundedRect(dc, x-4, y, itemWidth+8, candidateRowHeight, 4*scale)
			dc.Fill()
		}

		var textX float64
		if isTextIndex {
			// Text style: draw plain colored number (slightly larger font)
			indexStr := string(rune('0' + cand.Index))
			if indexTextFace != nil {
				dc.SetFontFace(indexTextFace)
				dc.SetColor(cfg.IndexColor)
				dc.DrawString(indexStr, x, candY+indexTextSize/3)
			}
			textX = x + indexTextWidths[i] + indexMargin
		} else {
			// Circle style: draw filled circle + white number
			indexX := x + indexSize/2
			dc.SetColor(cfg.IndexBgColor)
			dc.DrawCircle(indexX, candY, indexSize/2)
			dc.Fill()

			if smallFace != nil {
				dc.SetFontFace(smallFace)
				dc.SetColor(cfg.IndexColor)
				indexStr := string(rune('0' + cand.Index))
				tw, _ := dc.MeasureString(indexStr)
				dc.DrawString(indexStr, indexX-tw/2, candY+cfg.IndexFontSize/3)
			}
			textX = x + indexSize + indexMargin
		}

		// Draw candidate text
		if mainFace != nil {
			dc.SetFontFace(mainFace)
			dc.SetColor(cfg.TextColor)
			dc.DrawString(cand.Text, textX, candY+cfg.FontSize/3)
		}

		// Draw comment if present (slightly larger font for text-style index)
		if cand.Comment != "" && commentFace != nil {
			commentX := textX + measures[i].textWidth + 6*scale
			dc.SetFontFace(commentFace)
			dc.SetColor(r.getCommentColor())
			dc.DrawString(cand.Comment, commentX, candY+commentSize/3)
		}

		// Move to next item
		x += itemWidth + itemSpacing
	}

	// Draw page info with clickable page buttons (right aligned)
	if totalPages > 1 && pageFace != nil {
		dc.SetFontFace(pageFace)

		pageText := fmt.Sprintf(" %d/%d ", page, totalPages)
		pageW, _ := dc.MeasureString(pageText)

		totalW := arrowW + pageW + arrowW
		startX := width - cfg.Padding - totalW

		// Page up button rect
		pageUpBtnRect := CandidateRect{
			X: startX,
			Y: y,
			W: arrowW,
			H: candidateRowHeight,
		}

		// Draw page up hover background
		if hoverPageBtn == "up" {
			dc.SetColor(cfg.HoverBgColor)
			r.drawRoundedRect(dc, pageUpBtnRect.X, pageUpBtnRect.Y, pageUpBtnRect.W, pageUpBtnRect.H, 4*scale)
			dc.Fill()
		}

		// Draw left chevron (page up)
		leftArrowColor := cfg.IndexBgColor
		if page <= 1 {
			leftArrowColor = cfg.InputTextColor
		}
		leftCenterX := startX + arrowW/2
		dc.SetColor(leftArrowColor)
		r.drawChevronLeft(dc, leftCenterX, candY, arrowSize, 1.5*scale)
		result.PageUpRect = &pageUpBtnRect

		// Draw page text
		dc.SetColor(cfg.InputTextColor)
		dc.DrawString(pageText, startX+arrowW, candY+6*scale)

		// Page down button rect
		pageDownBtnRect := CandidateRect{
			X: startX + arrowW + pageW,
			Y: y,
			W: arrowW,
			H: candidateRowHeight,
		}

		// Draw page down hover background
		if hoverPageBtn == "down" {
			dc.SetColor(cfg.HoverBgColor)
			r.drawRoundedRect(dc, pageDownBtnRect.X, pageDownBtnRect.Y, pageDownBtnRect.W, pageDownBtnRect.H, 4*scale)
			dc.Fill()
		}

		// Draw right chevron (page down)
		rightArrowColor := cfg.IndexBgColor
		if page >= totalPages {
			rightArrowColor = cfg.InputTextColor
		}
		rightCenterX := startX + arrowW + pageW + arrowW/2
		dc.SetColor(rightArrowColor)
		r.drawChevronRight(dc, rightCenterX, candY, arrowSize, 1.5*scale)
		result.PageDownRect = &pageDownBtnRect
	}

	return dc.Image().(*image.RGBA), result
}
