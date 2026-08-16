# Blockpic

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

## Documentation

The project documentation is generated using Bazel and is available in PDF and HTML formats.

- **PDF**: `bazel build //ai:book_pdf`
- **HTML**: `bazel build //ai:html_chunked`

## Implementation

Implemented in Go using the `draw2d` library for vector graphics.
