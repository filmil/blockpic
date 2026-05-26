// SPDX-License-Identifier: Apache-2.0
package renderer

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"github.com/filmil/hodos/pkg/ast"
	"github.com/filmil/hodos/pkg/shapes"
	"github.com/filmil/hodos/pkg/router"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
)

var (
	ColorPrimitive = color.RGBA{0xE6, 0xF2, 0xFF, 0xFF} // #E6F2FF
	ColorOutline   = color.RGBA{0x33, 0x33, 0x33, 0xFF} // #333333
	ColorText      = color.RGBA{0x00, 0x00, 0x00, 0xFF} // #000000
)

type Renderer struct {
	Width, Height int
}

func New(w, h int) *Renderer {
	return &Renderer{Width: w, Height: h}
}

func (r *Renderer) Draw(gc draw2d.GraphicContext, diag *ast.Diagram) error {
	gc.SetFillColor(color.White)
	gc.Clear()

	gc.SetFontData(draw2d.FontData{Name: "luxisr", Family: draw2d.FontFamilySans, Style: draw2d.FontStyleNormal})
	gc.SetFontSize(10)

	layoutMap := make(map[string]ast.LayoutEntry)
	for _, l := range diag.Layout {
		layoutMap[l.Block] = l
	}

	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Name]
		if !ok {
			continue
		}

		w, h := r.CalculateSize(gc, b)

		gc.SetFillColor(ColorPrimitive)
		gc.SetStrokeColor(ColorOutline)
		gc.SetLineWidth(1)

		kind := strings.ToUpper(b.Kind)
		isMSI := kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" || kind == "REG" || kind == "CTR" || kind == "SRG"

		if isMSI {
			shapes.DrawMSIBlock(gc, l.X, l.Y, w, h)
		} else {
			switch kind {
			case "AND":
				shapes.DrawAND(gc, l.X, l.Y, w, h)
			case "OR":
				shapes.DrawOR(gc, l.X, l.Y, w, h)
			case "NOT":
				shapes.DrawNOT(gc, l.X, l.Y, w, h)
			case "XOR":
				shapes.DrawXOR(gc, l.X, l.Y, w, h)
			case "FLIPFLOP", "DFF", "JKFF", "TFF", "SRFF":
				shapes.DrawDFF(gc, l.X, l.Y, w, h)
			case "MUX":
				shapes.DrawMUX(gc, l.X, l.Y, w, h)
			case "DEMUX":
				shapes.DrawDEMUX(gc, l.X, l.Y, w, h)
			case "VCC":
				shapes.DrawVCC(gc, l.X, l.Y, w, h)
			case "GND":
				shapes.DrawGND(gc, l.X, l.Y, w, h)
			case "CLOCK":
				shapes.DrawClock(gc, l.X, l.Y, w, h)
			default:
				gc.BeginPath()
				gc.MoveTo(l.X, l.Y)
				gc.LineTo(l.X+w, l.Y)
				gc.LineTo(l.X+w, l.Y+h)
				gc.LineTo(l.X, l.Y+h)
				gc.Close()
				gc.FillStroke()
			}
		}

		gc.SetFillColor(ColorText)
		renderText(gc, b.Name, l.X+2, l.Y-5)

		if isMSI {
			// Center the MSI kind label in the common control block
			renderText(gc, kind, l.X+w/2-float64(len(kind)*3), l.Y+h*0.15+4)
		}

		r.renderPins(gc, b, l, w, h, isMSI)
	}

	for _, w := range diag.Wires {
		srcL, ok1 := layoutMap[w.Source.Block]
		dstL, ok2 := layoutMap[w.Destination.Block]
		if !ok1 || !ok2 {
			continue
		}

		var srcB, dstB ast.Block
		for _, b := range diag.Blocks {
			if b.Name == w.Source.Block {
				srcB = b
			}
			if b.Name == w.Destination.Block {
				dstB = b
			}
		}
		sw, sh := r.CalculateSize(gc, srcB)
		dw, dh := r.CalculateSize(gc, dstB)

		x1, y1, d1 := r.getPinPos(srcB, w.Source.Pin, srcL, sw, sh, false)
		x2, y2, d2 := r.getPinPos(dstB, w.Destination.Pin, dstL, dw, dh, true)

		points := router.Route(x1, y1, d1, x2, y2, d2, w.Hint)

		gc.SetStrokeColor(ColorOutline)
		thickness := float64(w.Thickness)
		if thickness <= 0 {
			thickness = 1
		}
		gc.SetLineWidth(thickness)
		
		gc.BeginPath()
		if len(points) > 0 {
			gc.MoveTo(points[0].X, points[0].Y)
			for i := 1; i < len(points); i++ {
				gc.LineTo(points[i].X, points[i].Y)
			}
		}
		gc.Stroke()

		if len(points) > 1 {
			last := points[len(points)-1]
			prev := points[len(points)-2]
			
			aw := 6.0
			ah := 4.0
			
			gc.BeginPath()
			gc.SetFillColor(ColorOutline)
			
			if last.X > prev.X { // pointing right
				gc.MoveTo(last.X, last.Y)
				gc.LineTo(last.X-aw, last.Y-ah/2)
				gc.LineTo(last.X-aw, last.Y+ah/2)
			} else if last.X < prev.X { // pointing left
				gc.MoveTo(last.X, last.Y)
				gc.LineTo(last.X+aw, last.Y-ah/2)
				gc.LineTo(last.X+aw, last.Y+ah/2)
			} else if last.Y > prev.Y { // pointing down
				gc.MoveTo(last.X, last.Y)
				gc.LineTo(last.X-ah/2, last.Y-aw)
				gc.LineTo(last.X+ah/2, last.Y-aw)
			} else if last.Y < prev.Y { // pointing up
				gc.MoveTo(last.X, last.Y)
				gc.LineTo(last.X-ah/2, last.Y+aw)
				gc.LineTo(last.X+ah/2, last.Y+aw)
			}
			
			gc.Close()
			gc.Fill()
		}

		if w.Crossline > 0 && len(points) > 1 {
			midIdx := len(points) / 2
			p1 := points[midIdx-1]
			p2 := points[midIdx]
			mx, my := (p1.X+p2.X)/2, (p1.Y+p2.Y)/2
			
			gc.SetLineWidth(1)
			gc.BeginPath()
			gc.MoveTo(mx-4, my+4)
			gc.LineTo(mx+4, my-4)
			gc.Stroke()
			
			gc.SetFillColor(ColorText)
			renderText(gc, fmt.Sprintf("%d", w.Crossline), mx+2, my-2)
		}
	}

	return nil
}

func (r *Renderer) Render(diag *ast.Diagram) (image.Image, error) {
	dest := image.NewRGBA(image.Rect(0, 0, r.Width, r.Height))
	gc := draw2dimg.NewGraphicContext(dest)
	err := r.Draw(gc, diag)
	return dest, err
}

func (r *Renderer) CalculateSize(gc draw2d.GraphicContext, b ast.Block) (float64, float64) {
	if b.Size != nil {
		return b.Size.W, b.Size.H
	}
	
	nameW, _ := measureText(gc, b.Name)
	
	maxInW, maxOutW := 0.0, 0.0
	for _, p := range b.Inputs {
		w, _ := measureText(gc, p.Name)
		if w > maxInW {
			maxInW = w
		}
	}
	for _, p := range b.Outputs {
		w, _ := measureText(gc, p.Name)
		if w > maxOutW {
			maxOutW = w
		}
	}

	minW := 60.0
	kind := strings.ToUpper(b.Kind)
	isMSI := kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" || kind == "REG" || kind == "CTR" || kind == "SRG"
	if isMSI {
		minW = 80.0
	}
	
	contentW := maxInW + maxOutW + 20.0
	if nameW+10 > contentW {
		contentW = nameW + 10
	}
	
	w := minW
	if contentW > w {
		w = contentW
	}
	
	numPins := len(b.Inputs)
	if len(b.Outputs) > numPins {
		numPins = len(b.Outputs)
	}
	if numPins < 2 {
		numPins = 2
	}
	
	h := float64(numPins) * 20.0
	if isMSI {
		h = float64(numPins)*25.0 + 30.0
	}
	
	// Add space for params if they exist
	if len(b.Params) > 0 {
	    // just rough estimate, we render it below name
	    h += float64(len(b.Params)) * 12.0
	}
	
	return w, h
}

func (r *Renderer) renderPins(gc draw2d.GraphicContext, b ast.Block, l ast.LayoutEntry, w, h float64, isMSI bool) {
	gc.SetFillColor(ColorText)
	offsetY := 0.0
	usableH := h
	if isMSI {
		offsetY = h * 0.3 // Push pins below the common control block
		usableH = h * 0.7
	}

	for i, p := range b.Inputs {
		pos := r.getPinPositionHint(p.Position, false)
		
		var px, py float64
		if pos == router.DirLeft {
			py = l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Inputs))+1)
			px = l.X
		} else if pos == router.DirTop {
			py = l.Y
			px = l.X + (float64(i)+1)*w/(float64(len(b.Inputs))+1)
		} else if pos == router.DirBottom {
			py = l.Y + h
			px = l.X + (float64(i)+1)*w/(float64(len(b.Inputs))+1)
		} else {
			// fallback left
			py = l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Inputs))+1)
			px = l.X
		}

		label := p.Name
		
		// Dependency Notation parsing
		if strings.HasPrefix(p.Name, "C") || strings.HasPrefix(p.Name, "CLK") {
			gc.BeginPath()
			if pos == router.DirLeft {
				gc.MoveTo(px, py-4)
				gc.LineTo(px+6, py)
				gc.LineTo(px, py+4)
			} else if pos == router.DirBottom {
				gc.MoveTo(px-4, py)
				gc.LineTo(px, py-6)
				gc.LineTo(px+4, py)
			} else if pos == router.DirTop {
				gc.MoveTo(px-4, py)
				gc.LineTo(px, py+6)
				gc.LineTo(px+4, py)
			}
			gc.Stroke()
			label = "C"
		} else if strings.HasPrefix(p.Name, "R") {
			label = "R"
		} else if strings.HasPrefix(p.Name, "EN") {
			label = "EN"
		}

		if pos == router.DirLeft {
			if isMSI {
				renderText(gc, label, px+8, py+4)
			} else {
				renderText(gc, label, px+2, py+4)
			}
			gc.BeginPath()
			gc.MoveTo(px-2, py)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirTop {
			renderText(gc, label, px-4, py+12)
			gc.BeginPath()
			gc.MoveTo(px, py-2)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirBottom {
			renderText(gc, label, px-4, py-6)
			gc.BeginPath()
			gc.MoveTo(px, py+2)
			gc.LineTo(px, py)
			gc.Stroke()
		}
	}

	for i, p := range b.Outputs {
		pos := r.getPinPositionHint(p.Position, true)

		var px, py float64
		if pos == router.DirRight {
			py = l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Outputs))+1)
			px = l.X + w
		} else if pos == router.DirTop {
			py = l.Y
			px = l.X + (float64(i)+1)*w/(float64(len(b.Outputs))+1)
		} else if pos == router.DirBottom {
			py = l.Y + h
			px = l.X + (float64(i)+1)*w/(float64(len(b.Outputs))+1)
		} else {
			// fallback right
			py = l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Outputs))+1)
			px = l.X + w
		}

		label := p.Name
		
		if pos == router.DirRight {
			if isMSI {
				renderText(gc, label, px-20, py+4)
			} else {
				renderText(gc, label, px-15, py+4)
			}
			gc.BeginPath()
			gc.MoveTo(px, py)
			gc.LineTo(px+2, py)
			gc.Stroke()
		} else if pos == router.DirTop {
			renderText(gc, label, px-4, py+12)
			gc.BeginPath()
			gc.MoveTo(px, py-2)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirBottom {
			renderText(gc, label, px-4, py-6)
			gc.BeginPath()
			gc.MoveTo(px, py+2)
			gc.LineTo(px, py)
			gc.Stroke()
		}
	}
}

func (r *Renderer) getPinPositionHint(hint string, isDest bool) router.Direction {
	switch hint {
	case "left":
		return router.DirLeft
	case "right":
		return router.DirRight
	case "top":
		return router.DirTop
	case "bottom":
		return router.DirBottom
	default:
		if isDest {
			return router.DirRight
		}
		return router.DirLeft
	}
}

func (r *Renderer) getPinPos(b ast.Block, pinName string, l ast.LayoutEntry, w, h float64, isDest bool) (float64, float64, router.Direction) {
	if pinName == "" {
		if isDest {
			return l.X, l.Y + h/2, router.DirLeft
		}
		return l.X + w, l.Y + h/2, router.DirRight
	}

	kind := strings.ToUpper(b.Kind)
	isMSI := kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" || kind == "REG" || kind == "CTR" || kind == "SRG"
	
	offsetY := 0.0
	usableH := h
	if isMSI {
		offsetY = h * 0.3
		usableH = h * 0.7
	}

	for i, p := range b.Inputs {
		if p.Name == pinName {
			pos := r.getPinPositionHint(p.Position, false)
			if pos == router.DirLeft {
				return l.X, l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Inputs))+1), pos
			} else if pos == router.DirTop {
				return l.X + (float64(i)+1)*w/(float64(len(b.Inputs))+1), l.Y, pos
			} else if pos == router.DirBottom {
				return l.X + (float64(i)+1)*w/(float64(len(b.Inputs))+1), l.Y + h, pos
			}
			return l.X, l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Inputs))+1), router.DirLeft
		}
	}
	for i, p := range b.Outputs {
		if p.Name == pinName {
			pos := r.getPinPositionHint(p.Position, true)
			if pos == router.DirRight {
				return l.X + w, l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Outputs))+1), pos
			} else if pos == router.DirTop {
				return l.X + (float64(i)+1)*w/(float64(len(b.Outputs))+1), l.Y, pos
			} else if pos == router.DirBottom {
				return l.X + (float64(i)+1)*w/(float64(len(b.Outputs))+1), l.Y + h, pos
			}
			return l.X + w, l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Outputs))+1), router.DirRight
		}
	}

	if isDest {
		return l.X, l.Y + h/2, router.DirLeft
	}
	return l.X + w, l.Y + h/2, router.DirRight
}

func measureText(gc draw2d.GraphicContext, text string) (float64, float64) {
	lines := strings.Split(strings.ReplaceAll(text, "\\n", "\n"), "\n")
	lineHeight := 12.0
	maxWidth := 0.0
	
	for _, line := range lines {
		cleanLine := strings.ReplaceAll(line, "~", "")
		cleanLine = strings.ReplaceAll(cleanLine, "_", "")
		l, _, r, _ := gc.GetStringBounds(cleanLine)
		w := r - l
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth, float64(len(lines)) * lineHeight
}

func renderText(gc draw2d.GraphicContext, text string, x, y float64) float64 {
	lines := strings.Split(strings.ReplaceAll(text, "\\n", "\n"), "\n")
	lineHeight := 12.0
	maxWidth := 0.0

	for i, line := range lines {
		ly := y + float64(i)*lineHeight
		cx := x
		
		inOver := false
		inUnder := false
		
		runes := []rune(line)
		buf := ""
		
		flush := func() {
			if len(buf) > 0 {
				gc.FillStringAt(buf, cx, ly)
				l, t, r, b := gc.GetStringBounds(buf)
				_, _ = t, b
				w := r - l
				
				if inOver {
					gc.BeginPath()
					gc.MoveTo(cx, ly-10)
					gc.LineTo(cx+w, ly-10)
					gc.Stroke()
				}
				if inUnder {
					gc.BeginPath()
					gc.MoveTo(cx, ly+2)
					gc.LineTo(cx+w, ly+2)
					gc.Stroke()
				}
				cx += w
				buf = ""
			}
		}
		
		for _, r := range runes {
			if r == '~' {
				flush()
				inOver = !inOver
			} else if r == '_' {
				flush()
				inUnder = !inUnder
			} else {
				buf += string(r)
			}
		}
		flush()
		if cx - x > maxWidth {
			maxWidth = cx - x
		}
	}
	return maxWidth
}
