// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"
	"io/ioutil"

	"github.com/filmil/hodos/pkg/layout"
	"github.com/filmil/hodos/pkg/parser"
	"github.com/filmil/hodos/pkg/renderer"
	"github.com/filmil/hodos/pkg/scanner"
	"github.com/filmil/hodos/pkg/validation"
	"github.com/llgcode/draw2d/draw2dimg"
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
	format := renderFlags.String("format", "png", "Output format (png only for now)")
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

	// Apply layout if missing
	layout.AutoLayout(diag)

	r := renderer.New(800, 600)
	img, err := r.Render(diag)
	if err != nil {
		fmt.Printf("Error rendering: %v\n", err)
		os.Exit(1)
	}

	if *format == "png" {
		err = draw2dimg.SaveToPngFile(*output, img)
		if err != nil {
			fmt.Printf("Error saving file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully rendered to %s\n", *output)
	} else {
		fmt.Printf("Unsupported format: %s\n", *format)
		os.Exit(1)
	}
}
