// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"
	"io/ioutil"
	"github.com/filmil/blockpic/pkg/parser"
	"github.com/filmil/blockpic/pkg/scanner"
	"github.com/filmil/blockpic/pkg/validation"
	"github.com/filmil/blockpic/pkg/yosys"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: blockpic [compile|render|layout|refine|import] <file>")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "compile":
		compileCmd()
	case "render":
		renderCmd()
	case "layout":
		layoutCmd()
	case "refine":
		refineCmd()
	case "import":
		importCmd()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func compileCmd() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: blockpic compile <file.blockpic>")
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
	format := renderFlags.String("format", "yosys-json", "Output format (yosys-json)")
	output := renderFlags.String("o", "output.json", "Output file")

	if len(os.Args) < 3 {
		fmt.Println("Usage: blockpic render <file.blockpic> [-format yosys-json] [-o output.json]")
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

	if *format == "yosys-json" {
		jsonBytes, err := yosys.Convert(diag)
		if err != nil {
			fmt.Printf("Error generating Yosys JSON: %v\n", err)
			os.Exit(1)
		}
		if err := ioutil.WriteFile(*output, jsonBytes, 0644); err != nil {
			fmt.Printf("Error saving file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully generated Yosys JSON to %s\n", *output)
	} else {
		fmt.Printf("Unsupported format: %s\n", *format)
		os.Exit(1)
	}
}
