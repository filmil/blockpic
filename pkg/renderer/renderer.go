// SPDX-License-Identifier: Apache-2.0
package renderer

import (
	"fmt"
	"image"
	"image/color"
	"math"
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

type routedWire struct {
	Wire ast.Wire
	Path []router.Point
}

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

		nameInside := false
		if !isMSI {
			switch kind {
			case "AND", "OR", "NOT", "XOR", "FLIPFLOP", "DFF", "JKFF", "TFF", "SRFF", "MUX", "DEMUX", "VCC", "GND", "CLOCK":
				// keep outside
			default:
				nameInside = true
			}
		}

		gc.SetFillColor(ColorText)
		if nameInside {
			nameW, nameH := measureText(gc, b.Name)
			if err := r.drawText(gc, b.Name, l.X+w/2-nameW/2, l.Y+h/2+nameH/2-2, &b, &l, w, h); err != nil {
				return err
			}
		} else {
			if err := r.drawText(gc, b.Name, l.X+2, l.Y-5, nil, nil, 0, 0); err != nil {
				return err
			}
		}

		if isMSI {
			// Center the MSI kind label in the common control block
			kindW, _ := measureText(gc, kind)
			if err := r.drawText(gc, kind, l.X+w/2-kindW/2, l.Y+h*0.15+4, &b, &l, w, h); err != nil {
				return err
			}
		}

		if err := r.renderPins(gc, b, l, w, h, isMSI); err != nil {
			return err
		}
	}
	var obstacles []router.Obstacle
	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Name]
		if !ok {
			continue
		}
		bw, bh := r.CalculateSize(gc, b)
		obstacles = append(obstacles, router.Obstacle{
			Name: b.Name,
			X:    l.X,
			Y:    l.Y,
			W:    bw,
			H:    bh,
		})

		kind := strings.ToUpper(b.Kind)
		isMSI := kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" || kind == "REG" || kind == "CTR" || kind == "SRG"
		nameInside := false
		if !isMSI {
			switch kind {
			case "AND", "OR", "NOT", "XOR", "FLIPFLOP", "DFF", "JKFF", "TFF", "SRFF", "MUX", "DEMUX", "VCC", "GND", "CLOCK":
				// keep outside
			default:
				nameInside = true
			}
		}

		if !nameInside {
			// Add block name text as obstacle
			nameW, nameH := measureText(gc, b.Name)
			obstacles = append(obstacles, router.Obstacle{
				Name: b.Name + "_label",
				X:    l.X + 2,
				Y:    l.Y - 5 - nameH,
				W:    nameW,
				H:    nameH,
			})
		}
	}

	var routedWires []routedWire

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

		points := router.Route(x1, y1, d1, x2, y2, d2, w.Hint, obstacles)
		routedWires = append(routedWires, routedWire{Wire: w, Path: points})

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
			if err := r.drawText(gc, fmt.Sprintf("%d", w.Crossline), mx+2, my-2, nil, nil, 0, 0); err != nil {
				return err
			}
		}
	}

	if err := r.verifyNoWireTextIntersections(gc, diag, layoutMap, routedWires); err != nil {
		return err
	}

	return nil
}

type textRect struct {
	Text string
	X, Y, W, H float64
}

func intersectsRect(x1, y1, x2, y2 float64, r textRect) bool {
	minX := math.Min(x1, x2)
	maxX := math.Max(x1, x2)
	minY := math.Min(y1, y2)
	maxY := math.Max(y1, y2)

	if x1 == x2 { // Vertical segment
		if x1 > r.X && x1 < r.X+r.W {
			// Check if the Y interval overlaps
			return !(maxY <= r.Y || minY >= r.Y+r.H)
		}
	} else if y1 == y2 { // Horizontal segment
		if y1 > r.Y && y1 < r.Y+r.H {
			// Check if the X interval overlaps
			return !(maxX <= r.X || minX >= r.X+r.W)
		}
	}
	return false
}

func (r *Renderer) verifyNoWireTextIntersections(gc draw2d.GraphicContext, diag *ast.Diagram, layoutMap map[string]ast.LayoutEntry, routedWires []routedWire) error {
	var rects []textRect

	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Name]
		if !ok {
			continue
		}

		w, h := r.CalculateSize(gc, b)
		kind := strings.ToUpper(b.Kind)
		isMSI := kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" || kind == "REG" || kind == "CTR" || kind == "SRG"

		// 1. Block name label
		nameW, nameH := measureText(gc, b.Name)
		nameInside := false
		if !isMSI {
			switch kind {
			case "AND", "OR", "NOT", "XOR", "FLIPFLOP", "DFF", "JKFF", "TFF", "SRFF", "MUX", "DEMUX", "VCC", "GND", "CLOCK":
				// keep outside
			default:
				nameInside = true
			}
		}

		if nameInside {
			rects = append(rects, textRect{
				Text: b.Name,
				X:    l.X + w/2 - nameW/2,
				Y:    l.Y + h/2 - nameH/2 - 2,
				W:    nameW,
				H:    nameH,
			})
		} else {
			rects = append(rects, textRect{
				Text: b.Name,
				X:    l.X + 2,
				Y:    l.Y - 5 - nameH,
				W:    nameW,
				H:    nameH,
			})
		}

		// 2. MSI kind label
		if isMSI {
			kindW, kindH := measureText(gc, kind)
			rects = append(rects, textRect{
				Text: kind,
				X:    l.X + w/2 - kindW/2,
				Y:    l.Y + h*0.15 + 4 - kindH,
				W:    kindW,
				H:    kindH,
			})
		}

		// 3. Pin labels
		offsetY := 0.0
		usableH := h
		if isMSI {
			offsetY = h * 0.3
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
				py = l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Inputs))+1)
				px = l.X
			}

			label := p.Name
			if strings.HasPrefix(p.Name, "C") || strings.HasPrefix(p.Name, "CLK") {
				label = "C"
			} else if strings.HasPrefix(p.Name, "R") {
				label = "R"
			} else if strings.HasPrefix(p.Name, "EN") {
				label = "EN"
			}

			tw, th := measureText(gc, label)
			var tx, ty float64
			if pos == router.DirLeft {
				if isMSI {
					tx = px + 8
				} else {
					tx = px + 2
				}
				ty = py + 4
			} else if pos == router.DirTop {
				tx = px - 4
				ty = py + 12
			} else if pos == router.DirBottom {
				tx = px - 4
				ty = py - 6
			}

			rects = append(rects, textRect{
				Text: label,
				X:    tx,
				Y:    ty - th,
				W:    tw,
				H:    th,
			})
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
				py = l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Outputs))+1)
				px = l.X + w
			}

			label := p.Name
			tw, th := measureText(gc, label)
			var tx, ty float64
			if pos == router.DirRight {
				if isMSI {
					tx = px - tw - 8
				} else {
					tx = px - tw - 4
				}
				ty = py + 4
			} else if pos == router.DirTop {
				tx = px - 4
				ty = py + 12
			} else if pos == router.DirBottom {
				tx = px - 4
				ty = py - 6
			}

			rects = append(rects, textRect{
				Text: label,
				X:    tx,
				Y:    ty - th,
				W:    tw,
				H:    th,
			})
		}
	}

	for _, rw := range routedWires {
		path := rw.Path
		for i := 0; i < len(path)-1; i++ {
			p1 := path[i]
			p2 := path[i+1]

			for _, r := range rects {
				if intersectsRect(p1.X, p1.Y, p2.X, p2.Y, r) {
					src := fmt.Sprintf("%s.%s", rw.Wire.Source.Block, rw.Wire.Source.Pin)
					dst := fmt.Sprintf("%s.%s", rw.Wire.Destination.Block, rw.Wire.Destination.Pin)
					return fmt.Errorf("wire from %s to %s intersects block text %q (bounding box: X=%.1f, Y=%.1f, W=%.1f, H=%.1f)", src, dst, r.Text, r.X, r.Y, r.W, r.H)
				}
			}
		}
	}

	return nil
}

func (r *Renderer) drawText(gc draw2d.GraphicContext, text string, x, y float64, b *ast.Block, l *ast.LayoutEntry, bw, bh float64) error {
	tw, th := measureText(gc, text)
	xMin := x
	xMax := x + tw
	yMin := y - th
	yMax := y

	// Check image bounds
	if xMin < 0 || xMax > float64(r.Width) || yMin < 0 || yMax > float64(r.Height) {
		return fmt.Errorf("text %q at (%.1f, %.1f) renders outside the image extents [0, 0, %d, %d]", text, x, y, r.Width, r.Height)
	}

	// Check block bounds if it is inside a block
	if b != nil && l != nil {
		bLeft := l.X
		bRight := l.X + bw
		bTop := l.Y
		bBottom := l.Y + bh
		tolerance := 0.0
		if xMin < bLeft-tolerance || xMax > bRight+tolerance || yMin < bTop-tolerance || yMax > bBottom+tolerance {
			fmt.Printf("FAIL TEXT BOUNDS: text=%q, xMin=%.2f, xMax=%.2f, yMin=%.2f, yMax=%.2f, bLeft=%.2f, bRight=%.2f, bTop=%.2f, bBottom=%.2f\n", text, xMin, xMax, yMin, yMax, bLeft, bRight, bTop, bBottom)
			return fmt.Errorf("text %q inside block %q renders outside its shape boundaries", text, b.Name)
		}
	}

	renderText(gc, text, x, y)
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

func (r *Renderer) renderPins(gc draw2d.GraphicContext, b ast.Block, l ast.LayoutEntry, w, h float64, isMSI bool) error {
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
				if err := r.drawText(gc, label, px+8, py+4, &b, &l, w, h); err != nil {
					return err
				}
			} else {
				if err := r.drawText(gc, label, px+2, py+4, &b, &l, w, h); err != nil {
					return err
				}
			}
			gc.BeginPath()
			gc.MoveTo(px-2, py)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirTop {
			if err := r.drawText(gc, label, px-4, py+12, &b, &l, w, h); err != nil {
				return err
			}
			gc.BeginPath()
			gc.MoveTo(px, py-2)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirBottom {
			if err := r.drawText(gc, label, px-4, py-6, &b, &l, w, h); err != nil {
				return err
			}
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
			tw, _ := measureText(gc, label)
			var startX float64
			if isMSI {
				startX = px - tw - 8
			} else {
				startX = px - tw - 4
			}
			if err := r.drawText(gc, label, startX, py+4, &b, &l, w, h); err != nil {
				return err
			}
			gc.BeginPath()
			gc.MoveTo(px, py)
			gc.LineTo(px+2, py)
			gc.Stroke()
		} else if pos == router.DirTop {
			if err := r.drawText(gc, label, px-4, py+12, &b, &l, w, h); err != nil {
				return err
			}
			gc.BeginPath()
			gc.MoveTo(px, py-2)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirBottom {
			if err := r.drawText(gc, label, px-4, py-6, &b, &l, w, h); err != nil {
				return err
			}
			gc.BeginPath()
			gc.MoveTo(px, py+2)
			gc.LineTo(px, py)
			gc.Stroke()
		}
	}
	return nil
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
