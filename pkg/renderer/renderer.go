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

func (r *Renderer) Render(diag *ast.Diagram) (image.Image, error) {
	dest := image.NewRGBA(image.Rect(0, 0, r.Width, r.Height))
	gc := draw2dimg.NewGraphicContext(dest)

	gc.SetFillColor(color.White)
	gc.Clear()

	layoutMap := make(map[string]ast.LayoutEntry)
	for _, l := range diag.Layout {
		layoutMap[l.Block] = l
	}

	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Name]
		if !ok {
			continue
		}

		w, h := r.calculateSize(b)

		gc.SetFillColor(ColorPrimitive)
		gc.SetStrokeColor(ColorOutline)
		gc.SetLineWidth(1)

		kind := strings.ToUpper(b.Kind)
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

		gc.SetFillColor(ColorText)
		gc.FillStringAt(b.Name, l.X+2, l.Y-5)

		if kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" {
			gc.FillStringAt(kind, l.X+5, l.Y+h/2+4)
		}

		r.renderPins(gc, b, l, w, h)
	}

	for _, w := range diag.Wires {
		srcL, ok1 := layoutMap[w.Source.Block]
		dstL, ok2 := layoutMap[w.Destination.Block]
		if !ok1 || !ok2 {
			continue
		}

		// Find block sizes for pin calculation
		var srcB, dstB ast.Block
		for _, b := range diag.Blocks {
			if b.Name == w.Source.Block {
				srcB = b
			}
			if b.Name == w.Destination.Block {
				dstB = b
			}
		}
		sw, sh := r.calculateSize(srcB)
		dw, dh := r.calculateSize(dstB)

		x1, y1 := r.getPinPos(srcB, w.Source.Pin, srcL, sw, sh, false)
		x2, y2 := r.getPinPos(dstB, w.Destination.Pin, dstL, dw, dh, true)

		points := router.Route(x1, y1, x2, y2, w.Hint)

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
			gc.FillStringAt(fmt.Sprintf("%d", w.Crossline), mx+2, my-2)
		}
	}

	return dest, nil
}

func (r *Renderer) calculateSize(b ast.Block) (float64, float64) {
	if b.Size != nil {
		return b.Size.W, b.Size.H
	}
	// Default size based on pin count
	numPins := len(b.Inputs)
	if len(b.Outputs) > numPins {
		numPins = len(b.Outputs)
	}
	if numPins < 2 {
		numPins = 2
	}
	return 60.0, float64(numPins) * 20.0
}

func (r *Renderer) renderPins(gc *draw2dimg.GraphicContext, b ast.Block, l ast.LayoutEntry, w, h float64) {
	gc.SetFillColor(ColorText)
	for i, p := range b.Inputs {
		py := l.Y + (float64(i)+1)*h/(float64(len(b.Inputs))+1)
		gc.FillStringAt(p.Name, l.X+2, py+4)
		gc.BeginPath()
		gc.MoveTo(l.X-2, py)
		gc.LineTo(l.X, py)
		gc.Stroke()
	}
	for i, p := range b.Outputs {
		py := l.Y + (float64(i)+1)*h/(float64(len(b.Outputs))+1)
		gc.FillStringAt(p.Name, l.X+w-15, py+4)
		gc.BeginPath()
		gc.MoveTo(l.X+w, py)
		gc.LineTo(l.X+w+2, py)
		gc.Stroke()
	}
}

func (r *Renderer) getPinPos(b ast.Block, pinName string, l ast.LayoutEntry, w, h float64, isDest bool) (float64, float64) {
	if pinName == "" {
		if isDest {
			return l.X, l.Y + h/2
		}
		return l.X + w, l.Y + h/2
	}

	for i, p := range b.Inputs {
		if p.Name == pinName {
			return l.X, l.Y + (float64(i)+1)*h/(float64(len(b.Inputs))+1)
		}
	}
	for i, p := range b.Outputs {
		if p.Name == pinName {
			return l.X + w, l.Y + (float64(i)+1)*h/(float64(len(b.Outputs))+1)
		}
	}

	return l.X + w/2, l.Y + h/2
}
