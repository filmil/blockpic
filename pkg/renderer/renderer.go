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

		w, h := 60.0, 40.0
		if b.Size != nil {
			w, h = b.Size.W, b.Size.H
		}

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
			// Default rectangle for REGISTER, COUNTER, etc.
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
		
		// Internal label if kind is special but use standard rect
		if kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" {
			gc.FillStringAt(kind, l.X+5, l.Y+h/2+4)
		}
	}

	for _, w := range diag.Wires {
		srcL, ok1 := layoutMap[w.Source.Block]
		dstL, ok2 := layoutMap[w.Destination.Block]
		if !ok1 || !ok2 {
			continue
		}

		x1, y1 := srcL.X+60, srcL.Y+20
		x2, y2 := dstL.X, dstL.Y+20

		points := router.Route(x1, y1, x2, y2, w.Hint)

		gc.SetStrokeColor(ColorOutline)
		gc.SetLineWidth(float64(w.Thickness))
		if w.Thickness == 0 {
			gc.SetLineWidth(1)
		}
		
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
