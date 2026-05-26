// SPDX-License-Identifier: Apache-2.0
package renderer

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
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

	// Stage 1: Draw all block shapes and pin indicators only, omitting text
	var blockObstacles []router.Obstacle
	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Label]
		if !ok {
			continue
		}

		w, h := r.CalculateSize(gc, b)
		blockObstacles = append(blockObstacles, router.Obstacle{
			Name: b.Label,
			X:    l.X,
			Y:    l.Y,
			W:    w,
			H:    h,
		})

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

		if err := r.renderPinGraphics(gc, b, l, w, h, isMSI); err != nil {
			return err
		}
	}

	// Stage 2: Route and draw wire lines and arrowheads
	var routedWires []routedWire

	type wireSortInfo struct {
		wire       ast.Wire
		x1, y1, x2, y2 float64
		d1, d2     router.Direction
		distance   float64
		isFeedback bool
	}

	var wireInfos []wireSortInfo
	for _, w := range diag.Wires {
		srcL, ok1 := layoutMap[w.Source.Block]
		dstL, ok2 := layoutMap[w.Destination.Block]
		if !ok1 || !ok2 {
			continue
		}

		var srcB, dstB ast.Block
		for _, b := range diag.Blocks {
			if b.Label == w.Source.Block {
				srcB = b
			}
			if b.Label == w.Destination.Block {
				dstB = b
			}
		}
		sw, sh := r.CalculateSize(gc, srcB)
		dw, dh := r.CalculateSize(gc, dstB)

		x1, y1, d1 := r.getPinPos(srcB, w.Source.Pin, srcL, sw, sh, false)
		x2, y2, d2 := r.getPinPos(dstB, w.Destination.Pin, dstL, dw, dh, true)

		distance := math.Abs(x2-x1) + math.Abs(y2-y1)
		isFeedback := x2 < x1

		wireInfos = append(wireInfos, wireSortInfo{
			wire:       w,
			x1:         x1,
			y1:         y1,
			x2:         x2,
			y2:         y2,
			d1:         d1,
			d2:         d2,
			distance:   distance,
			isFeedback: isFeedback,
		})
	}

	var feedforward []wireSortInfo
	var feedback []wireSortInfo

	for _, info := range wireInfos {
		if info.isFeedback {
			feedback = append(feedback, info)
		} else {
			feedforward = append(feedforward, info)
		}
	}

	sort.Slice(feedforward, func(i, j int) bool {
		if math.Abs(feedforward[i].distance-feedforward[j].distance) > 0.001 {
			return feedforward[i].distance < feedforward[j].distance
		}
		return feedforward[i].wire.Source.Block < feedforward[j].wire.Source.Block
	})
	sort.Slice(feedback, func(i, j int) bool {
		if math.Abs(feedback[i].distance-feedback[j].distance) > 0.001 {
			return feedback[i].distance < feedback[j].distance
		}
		return feedback[i].wire.Source.Block < feedback[j].wire.Source.Block
	})

	var sortedWires []wireSortInfo
	sortedWires = append(sortedWires, feedforward...)
	sortedWires = append(sortedWires, feedback...)

	var routedSegments []router.Segment

	for _, info := range sortedWires {
		// Route wire using blockObstacles and already routed segments
		points := router.Route(info.x1, info.y1, info.d1, info.x2, info.y2, info.d2, info.wire.Hint, blockObstacles, routedSegments)
		routedWires = append(routedWires, routedWire{Wire: info.wire, Path: points})

		for i := 0; i < len(points)-1; i++ {
			routedSegments = append(routedSegments, router.Segment{
				P1: points[i],
				P2: points[i+1],
			})
		}

		gc.SetStrokeColor(ColorOutline)
		thickness := float64(info.wire.Thickness)
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

		if info.wire.Crossline > 0 && len(points) > 1 {
			midIdx := len(points) / 2
			p1 := points[midIdx-1]
			p2 := points[midIdx]
			mx, my := (p1.X+p2.X)/2, (p1.Y+p2.Y)/2
			
			gc.SetLineWidth(1)
			gc.BeginPath()
			gc.MoveTo(mx-4, my+4)
			gc.LineTo(mx+4, my-4)
			gc.Stroke()
		}
	}

	// Stage 3: Collect, layout, and draw text labels
	labels := r.collectTextLabels(gc, diag, layoutMap, routedWires)
	placedRects, err := r.layoutAndDrawText(gc, labels, blockObstacles, routedWires)
	if err != nil {
		return err
	}

	if err := r.verifyNoWireTextIntersections(placedRects, routedWires); err != nil {
		return err
	}

	return nil
}

type textLabel struct {
	text       string
	itemType   string // "name_inside", "name_outside", "msi_kind", "pin", "crossline"
	ownerLabel string // block label
	block      *ast.Block
	layout     *ast.LayoutEntry
	bw, bh     float64
	defaultX   float64
	defaultY   float64
	// For crosslines
	isHoriz    bool
	midPoint   router.Point
}

func (r *Renderer) renderPinGraphics(gc draw2d.GraphicContext, b ast.Block, l ast.LayoutEntry, w, h float64, isMSI bool) error {
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
		}

		if pos == router.DirLeft {
			gc.BeginPath()
			gc.MoveTo(px-2, py)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirTop {
			gc.BeginPath()
			gc.MoveTo(px, py-2)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirBottom {
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
			py = l.Y + offsetY + (float64(i)+1)*usableH/(float64(len(b.Outputs))+1)
			px = l.X + w
		}

		if pos == router.DirRight {
			gc.BeginPath()
			gc.MoveTo(px, py)
			gc.LineTo(px+2, py)
			gc.Stroke()
		} else if pos == router.DirTop {
			gc.BeginPath()
			gc.MoveTo(px, py-2)
			gc.LineTo(px, py)
			gc.Stroke()
		} else if pos == router.DirBottom {
			gc.BeginPath()
			gc.MoveTo(px, py+2)
			gc.LineTo(px, py)
			gc.Stroke()
		}
	}
	return nil
}

func (r *Renderer) collectTextLabels(gc draw2d.GraphicContext, diag *ast.Diagram, layoutMap map[string]ast.LayoutEntry, routedWires []routedWire) []textLabel {
	var labels []textLabel

	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Label]
		if !ok {
			continue
		}
		w, h := r.CalculateSize(gc, b)
		kind := strings.ToUpper(b.Kind)
		isMSI := kind == "REGISTER" || kind == "COUNTER" || kind == "SHIFT_REGISTER" || kind == "REG" || kind == "CTR" || kind == "SRG"

		displayName := b.Name
		if displayName == "" {
			displayName = b.Label
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

		// 1. Block name label
		nameW, nameH := measureText(gc, displayName)
		blockRef := b
		layoutRef := l
		if nameInside {
			labels = append(labels, textLabel{
				text:       displayName,
				itemType:   "name_inside",
				ownerLabel: b.Label,
				block:      &blockRef,
				layout:     &layoutRef,
				bw:         w,
				bh:         h,
				defaultX:   l.X + w/2 - nameW/2,
				defaultY:   l.Y + h/2 + nameH/2 - 2,
			})
		} else {
			labels = append(labels, textLabel{
				text:       displayName,
				itemType:   "name_outside",
				ownerLabel: b.Label,
				block:      &blockRef,
				layout:     &layoutRef,
				bw:         w,
				bh:         h,
				defaultX:   l.X + 2,
				defaultY:   l.Y - 5,
			})
		}

		// 2. MSI kind label
		if isMSI {
			kindW, _ := measureText(gc, kind)
			labels = append(labels, textLabel{
				text:       kind,
				itemType:   "msi_kind",
				ownerLabel: b.Label,
				block:      &blockRef,
				layout:     &layoutRef,
				bw:         w,
				bh:         h,
				defaultX:   l.X + w/2 - kindW/2,
				defaultY:   l.Y + h*0.15 + 4,
			})
		}

		// 3. Pin labels
		offsetY := 0.0
		usableH := h
		if isMSI {
			offsetY = h * 0.3
			usableH = h * 0.7
		}

		showPins := true
		if isBasicElement(b.Kind) {
			showPins = b.Params["show_port_names"] == "true"
		}

		if showPins {
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

				pinLabel := p.Name
				if strings.HasPrefix(p.Name, "C") || strings.HasPrefix(p.Name, "CLK") {
					pinLabel = "C"
				} else if strings.HasPrefix(p.Name, "R") {
					pinLabel = "R"
				} else if strings.HasPrefix(p.Name, "EN") {
					pinLabel = "EN"
				}

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

				labels = append(labels, textLabel{
					text:       pinLabel,
					itemType:   "pin",
					ownerLabel: b.Label,
					block:      &blockRef,
					layout:     &layoutRef,
					bw:         w,
					bh:         h,
					defaultX:   tx,
					defaultY:   ty,
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

				pinLabel := p.Name
				tw, _ := measureText(gc, pinLabel)
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

				labels = append(labels, textLabel{
					text:       pinLabel,
					itemType:   "pin",
					ownerLabel: b.Label,
					block:      &blockRef,
					layout:     &layoutRef,
					bw:         w,
					bh:         h,
					defaultX:   tx,
					defaultY:   ty,
				})
			}
		}
	}

	// 4. Wire crossline labels
	for _, rw := range routedWires {
		w := rw.Wire
		points := rw.Path
		if w.Crossline > 0 && len(points) > 1 {
			midIdx := len(points) / 2
			p1 := points[midIdx-1]
			p2 := points[midIdx]
			mx, my := (p1.X+p2.X)/2, (p1.Y+p2.Y)/2
			isHoriz := math.Abs(p1.Y-p2.Y) < 0.1

			labels = append(labels, textLabel{
				text:     fmt.Sprintf("%d", w.Crossline),
				itemType: "crossline",
				defaultX: mx + 2,
				defaultY: my - 2,
				isHoriz:  isHoriz,
				midPoint: router.Point{X: mx, Y: my},
			})
		}
	}

	return labels
}

func (r *Renderer) layoutAndDrawText(gc draw2d.GraphicContext, labels []textLabel, blockObstacles []router.Obstacle, routedWires []routedWire) ([]textRect, error) {
	var placedRects []textRect

	for _, item := range labels {
		tw, th := measureText(gc, item.text)
		
		// Generate candidate positions
		var candidates []struct{ x, y float64 }

		if item.itemType == "name_outside" {
			l := item.layout
			bw, bh := item.bw, item.bh
			candidates = []struct{ x, y float64 }{
				{l.X + 2, l.Y - 5},                     // Top (Default)
				{l.X + 2, l.Y + bh + 5 + th},           // Bottom
				{l.X - tw - 5, l.Y + bh/2 + th/2},      // Left
				{l.X + bw + 5, l.Y + bh/2 + th/2},      // Right
			}
		} else if item.itemType == "crossline" {
			mx, my := item.midPoint.X, item.midPoint.Y
			if item.isHoriz {
				candidates = []struct{ x, y float64 }{
					{mx - tw/2, my - 4},             // Top-centered
					{mx - tw/2, my + th + 4},         // Bottom-centered
					{mx + 2, my - 2},                 // Top-Right (Default fallback)
					{mx + 2, my + th + 2},             // Bottom-Right
				}
			} else {
				candidates = []struct{ x, y float64 }{
					{mx + 4, my + th/2},             // Right-centered
					{mx - tw - 4, my + th/2},         // Left-centered
					{mx + 2, my - 2},                 // Top-Right (Default fallback)
					{mx - tw - 2, my - 2},             // Top-Left
				}
			}
		} else {
			candidates = []struct{ x, y float64 }{
				{item.defaultX, item.defaultY},
			}
		}

		found := false
		var finalX, finalY float64

		for _, cand := range candidates {
			if !r.checkCollision(cand.x, cand.y, tw, th, item.itemType, item.ownerLabel, blockObstacles, routedWires, placedRects) {
				finalX, finalY = cand.x, cand.y
				found = true
				break
			}
		}

		if !found {
			finalX, finalY = candidates[0].x, candidates[0].y
		}

		if (item.itemType == "name_inside" || item.itemType == "msi_kind" || item.itemType == "pin") && item.block != nil && item.layout != nil {
			bLeft := item.layout.X
			bRight := item.layout.X + item.bw
			bTop := item.layout.Y
			bBottom := item.layout.Y + item.bh

			if finalX < bLeft {
				finalX = bLeft
			}
			if finalX+tw > bRight {
				finalX = bRight - tw
			}
			if finalX < bLeft {
				finalX = bLeft
			}

			if finalY-th < bTop {
				finalY = bTop + th
			}
			if finalY > bBottom {
				finalY = bBottom
			}
		}

		if (item.itemType == "name_inside" || item.itemType == "msi_kind" || item.itemType == "pin") && item.block != nil && item.layout != nil {
			bLeft := item.layout.X
			bRight := item.layout.X + item.bw
			bTop := item.layout.Y
			bBottom := item.layout.Y + item.bh
			tolerance := 0.001
			if finalX < bLeft-tolerance || finalX+tw > bRight+tolerance || finalY-th < bTop-tolerance || finalY > bBottom+tolerance {
				return nil, fmt.Errorf("text %q inside block %q renders outside its shape boundaries", item.text, item.ownerLabel)
			}
		}

		if finalX < 0 || finalX+tw > float64(r.Width) || finalY-th < 0 || finalY > float64(r.Height) {
			return nil, fmt.Errorf("text %q at (%.1f, %.1f) renders outside the image extents [0, 0, %d, %d]", item.text, finalX, finalY, r.Width, r.Height)
		}

		gc.SetFillColor(ColorText)
		renderText(gc, item.text, finalX, finalY)

		placedRects = append(placedRects, textRect{
			Text: item.text,
			X:    finalX,
			Y:    finalY - th,
			W:    tw,
			H:    th,
		})
	}

	return placedRects, nil
}

func (r *Renderer) checkCollision(x, y, tw, th float64, itemType string, ownerLabel string, blockObstacles []router.Obstacle, routedWires []routedWire, placedRects []textRect) bool {
	rect := textRect{
		X: x,
		Y: y - th,
		W: tw,
		H: th,
	}

	if rect.X < 0 || rect.X+rect.W > float64(r.Width) || rect.Y < 0 || rect.Y+rect.H > float64(r.Height) {
		return true
	}

	for _, obs := range blockObstacles {
		if itemType == "name_inside" || itemType == "msi_kind" || itemType == "pin" {
			if obs.Name == ownerLabel {
				tolerance := 0.001
				contained := rect.X >= obs.X-tolerance &&
					rect.X+rect.W <= obs.X+obs.W+tolerance &&
					rect.Y >= obs.Y-tolerance &&
					rect.Y+rect.H <= obs.Y+obs.H+tolerance
				if !contained {
					return true
				}
				continue
			}
		}

		if rect.X < obs.X+obs.W && rect.X+rect.W > obs.X && rect.Y < obs.Y+obs.H && rect.Y+rect.H > obs.Y {
			return true
		}
	}

	for _, rw := range routedWires {
		path := rw.Path
		for i := 0; i < len(path)-1; i++ {
			p1 := path[i]
			p2 := path[i+1]
			if intersectsRect(p1.X, p1.Y, p2.X, p2.Y, rect) {
				return true
			}
		}
	}

	for _, pr := range placedRects {
		if rect.X < pr.X+pr.W && rect.X+rect.W > pr.X && rect.Y < pr.Y+pr.H && rect.Y+rect.H > pr.Y {
			return true
		}
	}

	return false
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

func (r *Renderer) verifyNoWireTextIntersections(placedRects []textRect, routedWires []routedWire) error {
	for _, rw := range routedWires {
		path := rw.Path
		for i := 0; i < len(path)-1; i++ {
			p1 := path[i]
			p2 := path[i+1]

			for _, rect := range placedRects {
				if intersectsRect(p1.X, p1.Y, p2.X, p2.Y, rect) {
					src := fmt.Sprintf("%s.%s", rw.Wire.Source.Block, rw.Wire.Source.Pin)
					dst := fmt.Sprintf("%s.%s", rw.Wire.Destination.Block, rw.Wire.Destination.Pin)
					return fmt.Errorf("wire from %s to %s intersects block text %q (bounding box: X=%.1f, Y=%.1f, W=%.1f, H=%.1f)", src, dst, rect.Text, rect.X, rect.Y, rect.W, rect.H)
				}
			}
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
	
	displayName := b.Name
	if displayName == "" {
		displayName = b.Label
	}
	nameW, _ := measureText(gc, displayName)
	
	maxInW, maxOutW := 0.0, 0.0
	showPins := true
	if isBasicElement(b.Kind) {
		showPins = b.Params["show_port_names"] == "true"
	}
	if showPins {
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
	    h += float64(len(b.Params)) * 12.0
	}
	
	return w, h
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

type textSegment struct {
	text  string
	over  bool
	under bool
}

func parseSegments(s string) []textSegment {
	var segments []textSegment
	runes := []rune(s)
	i := 0
	n := len(runes)

	for i < n {
		if runes[i] == '~' {
			// Find matching '~'
			end := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == '~' {
					end = j
					break
				}
			}
			if end != -1 {
				// Found matching pair!
				if end > i+1 {
					segments = append(segments, textSegment{
						text:  string(runes[i+1 : end]),
						over:  true,
						under: false,
					})
				}
				i = end + 1
				continue
			}
		} else if runes[i] == '_' {
			// Find matching '_'
			end := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == '_' {
					end = j
					break
				}
			}
			if end != -1 {
				// Found matching pair!
				if end > i+1 {
					segments = append(segments, textSegment{
						text:  string(runes[i+1 : end]),
						over:  false,
						under: true,
					})
				}
				i = end + 1
				continue
			}
		}

		// Otherwise, collect normal characters until next '~' or '_' that HAS a matching partner
		start := i
		for i < n {
			if runes[i] == '~' {
				// Check if this '~' has a matching partner later
				hasPartner := false
				for j := i + 1; j < n; j++ {
					if runes[j] == '~' {
						hasPartner = true
						break
					}
				}
				if hasPartner {
					break
				}
			} else if runes[i] == '_' {
				// Check if this '_' has a matching partner later
				hasPartner := false
				for j := i + 1; j < n; j++ {
					if runes[j] == '_' {
						hasPartner = true
						break
					}
				}
				if hasPartner {
					break
				}
			}
			i++
		}
		if i > start {
			segments = append(segments, textSegment{
				text:  string(runes[start:i]),
				over:  false,
				under: false,
			})
		}
	}
	return segments
}

func measureText(gc draw2d.GraphicContext, text string) (float64, float64) {
	lines := strings.Split(strings.ReplaceAll(text, "\\n", "\n"), "\n")
	lineHeight := 12.0
	maxWidth := 0.0
	
	for _, line := range lines {
		segments := parseSegments(line)
		lineW := 0.0
		for _, seg := range segments {
			l, _, r, _ := gc.GetStringBounds(seg.text)
			lineW += r - l
		}
		if lineW > maxWidth {
			maxWidth = lineW
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
		
		segments := parseSegments(line)
		for _, seg := range segments {
			if len(seg.text) > 0 {
				gc.FillStringAt(seg.text, cx, ly)
				l, t, r, b := gc.GetStringBounds(seg.text)
				_, _ = t, b
				w := r - l
				
				if seg.over {
					gc.BeginPath()
					gc.MoveTo(cx, ly-10)
					gc.LineTo(cx+w, ly-10)
					gc.Stroke()
				}
				if seg.under {
					gc.BeginPath()
					gc.MoveTo(cx, ly+2)
					gc.LineTo(cx+w, ly+2)
					gc.Stroke()
				}
				cx += w
			}
		}
		if cx - x > maxWidth {
			maxWidth = cx - x
		}
	}
	return maxWidth
}

func isBasicElement(kind string) bool {
	switch strings.ToUpper(kind) {
	case "AND", "OR", "NOT", "XOR", "FLIPFLOP", "DFF", "JKFF", "TFF", "SRFF", "MUX", "DEMUX", "VCC", "GND", "CLOCK":
		return true
	default:
		return false
	}
}
