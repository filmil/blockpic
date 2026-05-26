// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitfield/script"
)

type Figure struct {
	Name   string
	Page   int
	Y      int
	Height int
}

var figures = []Figure{
	{"figure_1_1", 9, 0, 0},
	{"figure_2_1", 15, 0, 0},
	{"figure_2_2", 17, 0, 0},
	{"figure_2_3", 19, 0, 0},
	{"figure_2_4", 20, 0, 0},
	{"figure_2_5", 22, 0, 0},
	{"figure_2_6", 26, 0, 0},
	{"figure_2_7", 27, 0, 0},
	{"figure_2_8", 28, 120, 690},
	{"figure_2_9", 28, 810, 690},
	{"figure_2_10", 29, 0, 0},
	{"figure_2_11", 30, 0, 0},
	{"figure_2_12", 31, 0, 0},
	{"figure_2_13", 32, 0, 0},
	{"figure_2_14", 33, 0, 0},
	{"figure_2_15", 35, 0, 0},
	{"figure_2_16", 36, 120, 690},
	{"figure_2_17", 36, 810, 690},
	{"figure_2_18", 37, 120, 690},
	{"figure_2_19", 37, 810, 690},
	{"figure_2_20", 38, 0, 0},
	{"figure_2_21", 40, 0, 0},
	{"figure_2_22", 41, 0, 0},
	{"figure_2_23", 42, 0, 0},
	{"figure_2_24", 43, 0, 0},
	{"figure_3_1", 46, 0, 0},
	{"figure_3_2", 47, 0, 0},
	{"figure_3_3", 48, 0, 0},
	{"figure_3_4", 50, 0, 0},
	{"figure_3_5", 51, 0, 0},
	{"figure_4_1", 53, 0, 0},
	{"figure_4_2", 54, 0, 0},
}

func main() {
	destDir := "third_party/xilinx"
	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create dest dir: %v\n", err)
		os.Exit(1)
	}

	for _, fig := range figures {
		srcFile := fmt.Sprintf("/tmp/extracted_images/page-%02d.png", fig.Page)
		destFile := filepath.Join(destDir, fig.Name+".png")

		y := fig.Y
		h := fig.Height
		if y == 0 && h == 0 {
			y = 120
			h = 1380
		}

		cmd := fmt.Sprintf("convert %s -crop 1275x%d+0+%d +repage -trim %s", srcFile, h, y, destFile)
		fmt.Printf("Extracting %s (Page %d) -> %s\n", fig.Name, fig.Page, destFile)

		_, err := script.Exec(cmd).Stdout()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error running command %q: %v\n", cmd, err)
			os.Exit(1)
		}
	}
	fmt.Println("All figures extracted successfully!")
}
