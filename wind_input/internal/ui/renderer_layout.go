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

	// First pass: draw all index circles and record bounding boxes
	for i := range candidates {
		itemY := y + float64(i)*cfg.ItemHeight
		indexX := cfg.Padding + 14*scale
		indexY := itemY + cfg.ItemHeight/2
		dc.SetColor(cfg.IndexBgColor)
		dc.DrawCircle(indexX, indexY, 11*scale)
		dc.Fill()

		// Record bounding rectangle
		result.Rects[i] = CandidateRect{
			Index: i,
			X:     cfg.Padding - 2,
			Y:     itemY,
			W:     width - cfg.Padding*2 + 2,
			H:     cfg.ItemHeight,
		}
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

		// Draw left arrow triangle (page up)
		leftArrowColor := cfg.IndexBgColor
		if page <= 1 {
			leftArrowColor = cfg.InputTextColor
		}
		leftCenterX := startX + arrowW/2
		dc.SetColor(leftArrowColor)
		dc.MoveTo(leftCenterX+arrowSize/2, centerY-arrowSize/2)
		dc.LineTo(leftCenterX-arrowSize/2, centerY)
		dc.LineTo(leftCenterX+arrowSize/2, centerY+arrowSize/2)
		dc.ClosePath()
		dc.Fill()
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

		// Draw right arrow triangle (page down)
		rightArrowColor := cfg.IndexBgColor
		if page >= totalPages {
			rightArrowColor = cfg.InputTextColor
		}
		rightCenterX := startX + arrowW + pageW + arrowW/2
		dc.SetColor(rightArrowColor)
		dc.MoveTo(rightCenterX-arrowSize/2, centerY-arrowSize/2)
		dc.LineTo(rightCenterX+arrowSize/2, centerY)
		dc.LineTo(rightCenterX-arrowSize/2, centerY+arrowSize/2)
		dc.ClosePath()
		dc.Fill()
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

	// Measure all candidates to calculate total width
	type candMeasure struct {
		textWidth    float64
		commentWidth float64
		totalWidth   float64 // Total width including index, text, comment
	}
	measures := make([]candMeasure, len(candidates))

	indexSize := 18.0 * scale   // Index circle size
	indexMargin := 4.0 * scale  // Margin between index and text
	itemSpacing := 12.0 * scale // Space between items

	tmpDc := gg.NewContext(1, 1)
	if mainFace != nil {
		tmpDc.SetFontFace(mainFace)
		for i, cand := range candidates {
			tw, _ := tmpDc.MeasureString(cand.Text)
			measures[i].textWidth = tw
		}
	}
	if smallFace != nil {
		tmpDc.SetFontFace(smallFace)
		for i, cand := range candidates {
			if cand.Comment != "" {
				cw, _ := tmpDc.MeasureString(cand.Comment)
				measures[i].commentWidth = cw
			}
		}
	}

	// Calculate total width for each candidate
	for i, cand := range candidates {
		measures[i].totalWidth = indexSize + indexMargin + measures[i].textWidth
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

	// Total width
	minWidth := 200.0 * scale
	contentWidth := cfg.Padding*2 + candidatesWidth + pageInfoWidth
	if inputWidth > 0 {
		contentWidth = cfg.Padding*2 + inputWidth
		if candidatesWidth+pageInfoWidth > inputWidth {
			contentWidth = cfg.Padding*2 + candidatesWidth + pageInfoWidth
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

	// Draw input area (preedit) - if not hidden
	if !cfg.HidePreedit && input != "" {
		dc.SetColor(cfg.InputBgColor)
		r.drawRoundedRect(dc, cfg.Padding, y, width-cfg.Padding*2-2, inputHeight, 4*scale)
		dc.Fill()

		if mainFace != nil {
			dc.SetFontFace(mainFace)
			textX := cfg.Padding + 8*scale
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
	x := cfg.Padding
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

		// Draw index circle
		indexX := x + indexSize/2
		dc.SetColor(cfg.IndexBgColor)
		dc.DrawCircle(indexX, candY, indexSize/2)
		dc.Fill()

		// Draw index number
		if smallFace != nil {
			dc.SetFontFace(smallFace)
			dc.SetColor(cfg.IndexColor)
			indexStr := string(rune('0' + cand.Index))
			tw, _ := dc.MeasureString(indexStr)
			dc.DrawString(indexStr, indexX-tw/2, candY+cfg.IndexFontSize/3)
		}

		// Draw candidate text
		textX := x + indexSize + indexMargin
		if mainFace != nil {
			dc.SetFontFace(mainFace)
			dc.SetColor(cfg.TextColor)
			dc.DrawString(cand.Text, textX, candY+cfg.FontSize/3)
		}

		// Draw comment if present
		if cand.Comment != "" && smallFace != nil {
			commentX := textX + measures[i].textWidth + 6*scale
			dc.SetFontFace(smallFace)
			dc.SetColor(r.getCommentColor())
			dc.DrawString(cand.Comment, commentX, candY+cfg.IndexFontSize/3)
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

		// Draw left arrow triangle (page up)
		leftArrowColor := cfg.IndexBgColor
		if page <= 1 {
			leftArrowColor = cfg.InputTextColor
		}
		leftCenterX := startX + arrowW/2
		dc.SetColor(leftArrowColor)
		dc.MoveTo(leftCenterX+arrowSize/2, candY-arrowSize/2)
		dc.LineTo(leftCenterX-arrowSize/2, candY)
		dc.LineTo(leftCenterX+arrowSize/2, candY+arrowSize/2)
		dc.ClosePath()
		dc.Fill()
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

		// Draw right arrow triangle (page down)
		rightArrowColor := cfg.IndexBgColor
		if page >= totalPages {
			rightArrowColor = cfg.InputTextColor
		}
		rightCenterX := startX + arrowW + pageW + arrowW/2
		dc.SetColor(rightArrowColor)
		dc.MoveTo(rightCenterX-arrowSize/2, candY-arrowSize/2)
		dc.LineTo(rightCenterX+arrowSize/2, candY)
		dc.LineTo(rightCenterX-arrowSize/2, candY+arrowSize/2)
		dc.ClosePath()
		dc.Fill()
		result.PageDownRect = &pageDownBtnRect
	}

	return dc.Image().(*image.RGBA), result
}
