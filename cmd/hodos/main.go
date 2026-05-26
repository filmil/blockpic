// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"
	"io/ioutil"
	"image"

	"github.com/filmil/hodos/pkg/ast"
	"github.com/filmil/hodos/pkg/layout"
	"github.com/filmil/hodos/pkg/parser"
	"github.com/filmil/hodos/pkg/renderer"
	"github.com/filmil/hodos/pkg/scanner"
	"github.com/filmil/hodos/pkg/validation"
	_ "github.com/filmil/hodos/third_party/draw2dfont"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dsvg"
	"github.com/llgcode/draw2d/draw2dpdf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: hodos [compile|render] <file>")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "compile":
		compileCmd()
	case "render":
		renderCmd()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func compileCmd() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: hodos compile <file.hodos>")
		os.Exit(1)
	}
	file := os.Args[2]
	content, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	s := scanner.New(string(content))
	p := parser.New(s)
	diag := p.ParseDiagram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range p.Errors() {
			fmt.Printf(" - %s\n", err)
		}
		os.Exit(1)
	}

	if err := validation.Validate(diag); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully compiled diagram: %s\n", diag.Name)
}

func renderCmd() {
	renderFlags := flag.NewFlagSet("render", flag.ExitOnError)
	format := renderFlags.String("format", "png", "Output format (png, svg, pdf)")
	output := renderFlags.String("o", "output.png", "Output file")

	if len(os.Args) < 3 {
		fmt.Println("Usage: hodos render <file.hodos> [-format png] [-o output.png]")
		os.Exit(1)
	}

	file := os.Args[2]
	renderFlags.Parse(os.Args[3:])

	content, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	s := scanner.New(string(content))
	p := parser.New(s)
	diag := p.ParseDiagram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range p.Errors() {
			fmt.Printf(" - %s\n", err)
		}
		os.Exit(1)
	}

	if err := validation.Validate(diag); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		os.Exit(1)
	}


	r := renderer.New(800, 600)

	// Create a dummy image and graphic context for measuring text sizes during layout
	dummyImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	dummyGc := draw2dimg.NewGraphicContext(dummyImg)
	dummyGc.SetFontData(draw2d.FontData{Name: "luxisr", Family: draw2d.FontFamilySans, Style: draw2d.FontStyleNormal})
	dummyGc.SetFontSize(10)

	// Apply layout if missing
	layout.AutoLayout(diag, func(b ast.Block) (float64, float64) {
		return r.CalculateSize(dummyGc, b)
	})

	// Find the bounds of the layout to size the image
	maxX := 800.0
	maxY := 600.0
	layoutMap := make(map[string]ast.LayoutEntry)
	for _, l := range diag.Layout {
		layoutMap[l.Block] = l
	}

	for _, b := range diag.Blocks {
		l, ok := layoutMap[b.Label]
		if !ok {
			continue
		}
		bw, bh := r.CalculateSize(dummyGc, b)
		if l.X+bw+40 > maxX {
			maxX = l.X + bw + 40
		}
		if l.Y+bh+40 > maxY {
			maxY = l.Y + bh + 40
		}
	}

	// Recreate renderer with correct size
	r = renderer.New(int(maxX), int(maxY))

	if *format == "png" {
		img, err := r.Render(diag)
		if err != nil {
			fmt.Printf("Error rendering: %v\n", err)
			os.Exit(1)
		}
		err = draw2dimg.SaveToPngFile(*output, img)
		if err != nil {
			fmt.Printf("Error saving file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully rendered to %s\n", *output)
	} else if *format == "svg" {
		svg := draw2dsvg.NewSvg()
		svg.Width = fmt.Sprintf("%dpx", r.Width)
		svg.Height = fmt.Sprintf("%dpx", r.Height)
		gc := draw2dsvg.NewGraphicContext(svg)
		if err := r.Draw(gc, diag); err != nil {
			fmt.Printf("Error rendering SVG: %v\n", err)
			os.Exit(1)
		}
		if err := draw2dsvg.SaveToSvgFile(*output, svg); err != nil {
			fmt.Printf("Error saving file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully rendered SVG to %s\n", *output)
	} else if *format == "pdf" {
		pdf := draw2dpdf.NewPdf("PORTRAIT", "mm", "A4")
		gc := draw2dpdf.NewGraphicContext(pdf)
		if err := r.Draw(gc, diag); err != nil {
			fmt.Printf("Error rendering PDF: %v\n", err)
			os.Exit(1)
		}
		if err := draw2dpdf.SaveToPdfFile(*output, pdf); err != nil {
			fmt.Printf("Error saving file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully rendered PDF to %s\n", *output)
	} else {
		fmt.Printf("Unsupported format: %s\n", *format)
		os.Exit(1)
	}
}
