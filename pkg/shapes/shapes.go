// SPDX-License-Identifier: Apache-2.0
package shapes

import (
	"math"
	"github.com/llgcode/draw2d"
)

func DrawAND(gc draw2d.GraphicContext, x, y, w, h float64) {
	r := w / 2
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+r, y)
	gc.ArcTo(x+r, y+r, r, r, -math.Pi/2, math.Pi)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.FillStroke()
}

func DrawOR(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.QuadCurveTo(x+w/4, y+h/2, x, y+h)
	gc.QuadCurveTo(x+w*0.8, y+h, x+w, y+h/2)
	gc.QuadCurveTo(x+w*0.8, y, x, y)
	gc.Close()
	gc.FillStroke()
}

func DrawNOT(gc draw2d.GraphicContext, x, y, w, h float64) {
	// Triangle
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+w-8, y+h/2)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.FillStroke()

	// Inversion bubble
	gc.BeginPath()
	gc.ArcTo(x+w-4, y+h/2, 4, 4, 0, 2*math.Pi)
	gc.Close()
	gc.Stroke()
}
