// SPDX-License-Identifier: Apache-2.0
package shapes

import (
	"math"
	"github.com/llgcode/draw2d"
)

func DrawAND(gc draw2d.GraphicContext, x, y, w, h float64) {
	r := h / 2
	straightW := w - r
	if straightW < 0 {
		straightW = 0
		r = w
	}
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+straightW, y)
	gc.ArcTo(x+straightW, y+r, r, r, -math.Pi/2, math.Pi)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.FillStroke()
}

func DrawOR(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x, y)
	// Top curve
	gc.QuadCurveTo(x+w*0.6, y, x+w, y+h/2)
	// Bottom curve
	gc.QuadCurveTo(x+w*0.6, y+h, x, y+h)
	// Left curve
	gc.QuadCurveTo(x+w*0.25, y+h/2, x, y)
	gc.Close()
	gc.FillStroke()
}

func DrawNOT(gc draw2d.GraphicContext, x, y, w, h float64) {
	// Standard NOT gate is a triangle
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+w, y+h/2)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.FillStroke()
	// Note: Bubble is drawn as a functional indicator on the pin in renderer.go, 
	// but currently the renderer doesn't draw the bubble for NOT gates.
	// To keep it simple and backwards compatible, we'll draw the bubble here.
	bubbleR := 4.0
	gc.BeginPath()
	gc.ArcTo(x+w+bubbleR, y+h/2, bubbleR, bubbleR, 0, 2*math.Pi)
	gc.Close()
	gc.Stroke()
}

func DrawXOR(gc draw2d.GraphicContext, x, y, w, h float64) {
	// First curve
	gc.BeginPath()
	gc.MoveTo(x-5, y)
	gc.QuadCurveTo(x+w*0.25-5, y+h/2, x-5, y+h)
	gc.Stroke()

	// Body same as OR
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.QuadCurveTo(x+w*0.6, y, x+w, y+h/2)
	gc.QuadCurveTo(x+w*0.6, y+h, x, y+h)
	gc.QuadCurveTo(x+w*0.25, y+h/2, x, y)
	gc.Close()
	gc.FillStroke()
}

func DrawDFF(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+w, y)
	gc.LineTo(x+w, y+h)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.FillStroke()
}

func DrawMUX(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+w, y+h*0.2)
	gc.LineTo(x+w, y+h*0.8)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.FillStroke()
}

func DrawDEMUX(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x, y+h*0.2)
	gc.LineTo(x+w, y)
	gc.LineTo(x+w, y+h)
	gc.LineTo(x, y+h*0.8)
	gc.Close()
	gc.FillStroke()
}

func DrawVCC(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x+w/2, y+h)
	gc.LineTo(x+w/2, y+h/2)
	gc.MoveTo(x, y+h/2)
	gc.LineTo(x+w, y+h/2)
	gc.Stroke()
}

func DrawGND(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x+w/2, y)
	gc.LineTo(x+w/2, y+h/2)
	gc.MoveTo(x, y+h/2)
	gc.LineTo(x+w, y+h/2)
	gc.MoveTo(x+w/4, y+h*0.75)
	gc.LineTo(x+w*0.75, y+h*0.75)
	gc.MoveTo(x+w*0.4, y+h)
	gc.LineTo(x+w*0.6, y+h)
	gc.Stroke()
}

func DrawClock(gc draw2d.GraphicContext, x, y, w, h float64) {
	gc.BeginPath()
	gc.MoveTo(x, y+h)
	gc.LineTo(x+w/4, y+h)
	gc.LineTo(x+w/4, y)
	gc.LineTo(x+w*0.75, y)
	gc.LineTo(x+w*0.75, y+h)
	gc.LineTo(x+w, y+h)
	gc.Stroke()
}

func DrawMSIBlock(gc draw2d.GraphicContext, x, y, w, h float64) {
	controlHeight := h * 0.3
	notch := 10.0

	// Draw the IEEE Common Control Block (wider top, narrower bottom)
	gc.BeginPath()
	gc.MoveTo(x-notch, y)
	gc.LineTo(x+w+notch, y)
	gc.LineTo(x+w+notch, y+controlHeight-notch)
	gc.LineTo(x+w, y+controlHeight-notch)
	gc.LineTo(x+w, y+h)
	gc.LineTo(x, y+h)
	gc.LineTo(x, y+controlHeight-notch)
	gc.LineTo(x-notch, y+controlHeight-notch)
	gc.Close()
	gc.FillStroke()

	// Separator line
	gc.BeginPath()
	gc.MoveTo(x, y+controlHeight)
	gc.LineTo(x+w, y+controlHeight)
	gc.Stroke()
}
