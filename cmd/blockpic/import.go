// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/filmil/blockpic/pkg/format"
	"github.com/filmil/blockpic/pkg/yosys"
)

// importCmd reads a Yosys JSON netlist and writes it as blockpic source:
//
//	blockpic import <netlist.json> [-top MODULE] [-o out.blockpic]
//
// -top names the module to draw; without it, the netlist's top module
// is drawn. It exits with status 1 when the netlist cannot be read or
// written.
func importCmd() {
	importFlags := flag.NewFlagSet("import", flag.ExitOnError)
	top := importFlags.String("top", "", "Module to draw (default: the netlist's top module)")
	output := importFlags.String("o", "output.blockpic", "Output blockpic file")

	if len(os.Args) < 3 {
		fmt.Println("Usage: blockpic import <netlist.json> [-top MODULE] [-o output.blockpic]")
		os.Exit(1)
	}

	file := os.Args[2]
	importFlags.Parse(os.Args[3:])

	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading netlist: %v\n", err)
		os.Exit(1)
	}

	diag, err := yosys.Import(content, *top)
	if err != nil {
		fmt.Printf("Error importing netlist: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, format.Format(diag), 0644); err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully imported %s to %s\n", diag.Name, *output)
}
