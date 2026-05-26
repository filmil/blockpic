// SPDX-License-Identifier: Apache-2.0
package renderer

import (
	"image"
	"image/color"
	"github.com/filmil/hodos/pkg/ast"
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

	// Set background
	gc.SetFillColor(color.White)
	gc.Clear()

	// Build a map for easy block lookup
	blockMap := make(map[string]ast.Block)
	for _, b := range diag.Blocks {
		blockMap[b.Name] = b
	}

	// Layout lookup
	layoutMap := make(map[string]ast.LayoutEntry)
	for _, l := range diag.Layout {
		layoutMap[l.Block] = l
	}

	// Render Blocks
	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Name]
		if !ok {
			continue // Skip if no layout (auto-layout is Phase 2)
		}

		w, h := 60.0, 40.0
		if b.Size != nil {
			w, h = b.Size.W, b.Size.H
		}

		// Draw block rectangle
		gc.SetFillColor(ColorPrimitive)
		gc.SetStrokeColor(ColorOutline)
		gc.SetLineWidth(1)
		gc.BeginPath()
		gc.MoveTo(l.X, l.Y)
		gc.LineTo(l.X+w, l.Y)
		gc.LineTo(l.X+w, l.Y+h)
		gc.LineTo(l.X, l.Y+h)
		gc.Close()
		gc.FillStroke()

		// Text: Name
		gc.SetFillColor(ColorText)
		gc.FillStringAt(b.Name, l.X+2, l.Y+12)
	}

	// Render Wires
	for _, w := range diag.Wires {
		srcL, ok1 := layoutMap[w.Source.Block]
		dstL, ok2 := layoutMap[w.Destination.Block]
		if !ok1 || !ok2 {
			continue
		}

		// Simplified: draw a straight line between centers for now
		// (Orthogonal routing is Task 004)
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
