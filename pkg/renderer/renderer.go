// SPDX-License-Identifier: Apache-2.0
package renderer

import (
	"image"
	"image/color"
	"strings"
	"github.com/filmil/hodos/pkg/ast"
	"github.com/filmil/hodos/pkg/shapes"
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

		switch strings.ToUpper(b.Kind) {
		case "AND":
			shapes.DrawAND(gc, l.X, l.Y, w, h)
		case "OR":
			shapes.DrawOR(gc, l.X, l.Y, w, h)
		case "NOT":
			shapes.DrawNOT(gc, l.X, l.Y, w, h)
		default:
			// Default rectangle
			gc.BeginPath()
			gc.MoveTo(l.X, l.Y)
			gc.LineTo(l.X+w, l.Y)
			gc.LineTo(l.X+w, l.Y+h)
			gc.LineTo(l.X, l.Y+h)
			gc.Close()
			gc.FillStroke()
		}

		gc.SetFillColor(ColorText)
		gc.FillStringAt(b.Name, l.X+2, l.Y-5) // Move name above the gate
	}

	for _, w := range diag.Wires {
		srcL, ok1 := layoutMap[w.Source.Block]
		dstL, ok2 := layoutMap[w.Destination.Block]
		if !ok1 || !ok2 {
			continue
		}

		gc.SetStrokeColor(ColorOutline)
		gc.SetLineWidth(float64(w.Thickness))
		if w.Thickness == 0 {
			gc.SetLineWidth(1)
		}
		
		gc.BeginPath()
		gc.MoveTo(srcL.X+30, srcL.Y+20)
		gc.LineTo(dstL.X+30, dstL.Y+20)
		gc.Stroke()
	}

	return dest, nil
}
