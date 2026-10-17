# Blockpic

Forgejo (canonical):
[![test](https://git.hdlfactory.com/HDL/blockpic/badges/workflows/test.yml/badge.svg?branch=main)](https://git.hdlfactory.com/HDL/blockpic/actions?workflow=test.yml)
[![samples](https://git.hdlfactory.com/HDL/blockpic/badges/workflows/samples.yml/badge.svg?branch=main)](https://git.hdlfactory.com/HDL/blockpic/actions?workflow=samples.yml)

GitHub (mirror, builds the book and docs):
[![CI](https://github.com/filmil/blockpic/actions/workflows/main.yml/badge.svg)](https://github.com/filmil/blockpic/actions/workflows/main.yml)
[![Deploy Docs](https://github.com/filmil/blockpic/actions/workflows/deploy_docs.yml/badge.svg)](https://github.com/filmil/blockpic/actions/workflows/deploy_docs.yml)
[![Release](https://github.com/filmil/blockpic/actions/workflows/release.yml/badge.svg)](https://github.com/filmil/blockpic/actions/workflows/release.yml)

Blockpic is a language for engineering block diagram typesetting, designed to produce diagrams with the aesthetics of Xilinx UG474 technical documentation.

## Usage

Blockpic provides a command-line interface for compiling and rendering diagrams.

### 1. Create a Diagram File
Create a file named `example.blockpic` with your logic definition:
```blockpic
diagram MyLogic {
    block AND1 { kind: "AND"; inputs: A, B [left]; outputs: O [right]; }
    block NOT1 { kind: "NOT"; inputs: I [left]; outputs: O [right]; }
    
    wire AND1.O -> NOT1.I { hint: "straight" }
}
```

### 2. Compile and Verify
Use the `compile` command to check for syntax errors:
```bash
bazel run //cmd/blockpic -- compile example.blockpic
```

### 3. Render the Diagram
Use the `render` command to generate an image (PNG is currently supported). Auto-layout will be applied if coordinates are not manually specified.
```bash
bazel run //cmd/blockpic -- render example.blockpic -format png -o output.png
```

### 4. Draw a Synthesized Netlist
The `import` command reads a Yosys JSON netlist and writes it as a
`.blockpic` file, which the steps above then render and refine. It reads,
for example, the netlist that `vivado_synthesis` in
[rules_openxc7](https://github.com/filmil/bazel_rules_openxc7) writes.
```bash
bazel run //cmd/blockpic -- import netlist.json -o design.blockpic
```
Without `-top`, the module that Yosys marked as the top is drawn.
Hidden cell names, such as `$abc$123$lut`, are replaced by the cell type and
a number, or by the name of the register bit that the cell holds.

### 5. Draw a Netlist with Bazel
In a Bazel build, `blockpic_netlist_diagram` in `//:defs.bzl` does steps 2
to 4 in one target. It takes any target whose only output is a Yosys JSON
netlist. A `vivado_synthesis` target of rules_openxc7 is one:
```python
load("@rules_openxc7//openxc7:defs.bzl", "vivado_project", "vivado_synthesis")
load("//:defs.bzl", "blockpic_netlist_diagram")

vivado_project(
    name = "counter",
    srcs = ["counter.v"],
    part = "xc7a200tfbg484-2",
    top_level = "counter",
)

vivado_synthesis(name = "counter_synth", project = ":counter")

blockpic_netlist_diagram(
    name = "counter_diagram",
    netlist = ":counter_synth",
    out = "counter.svg",
)
```
`bazel build //:counter_diagram` gives `counter.svg`. The imported
source is kept, as `counter_diagram.blockpic` from `//:counter_diagram_src`.
`tests/openxc7` is this example, with a test.

## Samples

`samples/` holds documents whose figures blockpic draws during the build.
`samples/blockpic-in-docs` is a three-page IEEE paper about a 4-bit LED
counter. One figure comes from a hand-written `.blockpic` file. The other
comes from the netlist that rules_openxc7 synthesizes from the Verilog.
```bash
bazel build //samples:pdfs
```
The PDFs are published by the `samples` workflow: every Sunday to the
rolling [`samples`](https://git.hdlfactory.com/HDL/blockpic/releases/tag/samples)
release, and to the release of every version tag.

## Documentation

The project documentation is generated using Bazel and is available in PDF and HTML formats.

- **PDF**: `bazel build //ai:book_pdf`
- **HTML**: `bazel build //ai:html_chunked`

## Implementation

Implemented in Go using the `draw2d` library for vector graphics.
