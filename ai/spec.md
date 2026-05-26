# Hodos Language Specification

Hodos is a domain-specific language (DSL) for engineering block diagram
typesetting, designed to produce diagrams with the aesthetics of Xilinx UG474
technical documentation.

## 1. Overview

Hodos focuses on clear, functional, and aesthetically minimalist diagrams. It
prioritizes orthogonal routing, precise port placement, and a standard color
palette for hardware logic primitives.

## 2. Core Concepts

### 2.1 Blocks and Primitives
A **Block** is the primary logical element. Each block has:
- **Name**: A unique identifier.
- **Kind**: The type of block (e.g., `AND`, `OR`, `FLIPFLOP`, `MUX`, `LUT`, `CUSTOM`).
- **Input Pins**: Named input ports.
- **Output Pins**: Named output ports.
- **Parameters**: Key-value pairs for configuration (e.g., `init: 0x1234`).
- **Dimensions**: Optional width and height.
- **Custom Rendering**: Support for custom rendering functions (Phase 2).

### 2.2 Pins (Ports)
Pins are the connection points on blocks.
- **Type**: `input` or `output`.
- **Position Hint**: `left`, `right`, `top`, `bottom`.

### 2.3 Wires
Wires connect an output pin of a source block to an input pin of a destination block.
- **Source**: `(block_name, pin_name)`.
- **Destination**: `(block_name, pin_name)`.
- **Routing Hints**: Hints for the router (e.g., `straight`, `L-shape`, `Z-shape`).
- **Bus Attributes**:
    - **Thickness**: Specifies if the wire is a bus and its relative thickness.
    - **Crossline**: Supports a crossline indicator (often used with a number to indicate bus width).

### 2.4 Aesthetics (UG474 Style)
- **Colors**:
    - Primitives: Light Blue (`#E6F2FF`) with a Dark Grey (`#333333`) outline.
    - Containers: Light Grey (`#F2F2F2`) or transparent with dashed borders.
    - Special Paths: Light Orange (`#FFF2E6`) for carry chains or shift registers.
- **Typography**: Sans-serif (Helvetica or Arial), 8pt to 10pt.
- **Line Weights**: 0.5pt for wires, 1.5pt for buses.

### 2.5 Shape Library and IEEE Standards
Hodos strictly adheres to **ANSI/IEEE Std 91-1984** for logic symbols. It supports both distinctive-shape and rectangular-shape symbols.

#### 2.5.1 Basic Logic Gates
- **Distinctive Shapes**: Standard semi-circular (AND), pointed (OR), and triangle (NOT) shapes.
- **Rectangular Shapes**: Uniform rectangles using internal qualifying symbols:
    - `&` : AND
    - `>=1`: OR
    - `1` : Buffer/Inverter
    - `=1`: XOR
    - `X/Y`: Specialized functions.

#### 2.5.2 Dependency Notation (for Complex Blocks)
For MSI components like Registers and Counters, Hodos supports IEEE dependency notation to represent internal logic relationships:
- **`G` (AND)**: Gate dependency.
- **`C` (Control)**: Timing/Clock dependency (often used with the `>` dynamic indicator for edge-triggering).
- **`S` (Set) / `R` (Reset)**: Asynchronous set and reset.
- **`EN` (Enable)**: Output enable.
- **`M` (Mode)**: Mode dependency.
- **`V` (OR)**: OR dependency.

#### 2.5.3 Functional Indicators
- **Bubble (`(o)`)**: External negation.
- **Wedge (`>`)**: Dynamic indicator for edge-triggered inputs.
- **Triangle (`(v)`)**: Three-state (Hi-Z) output indicator.

## 3. Syntax (Draft)

Hodos uses a custom DSL designed for readability.

```hodos
diagram MyLogic {
    config {
        style: "Xilinx";
        font: "Helvetica", 9pt;
    }

    // Block declarations
    block LUT6_A {
        kind: "LUT";
        params: { init: "0x8000000000000000" };
        inputs: I0, I1, I2, I3, I4, I5;
        outputs: O;
        size: (40, 60);
    }

    block DFF_1 {
        kind: "FLIPFLOP";
        inputs: D, CLK [bottom], CE [left];
        outputs: Q;
    }

    // Wire declarations
    wire LUT6_A.O -> DFF_1.D {
        hint: "Z-shape";
    }

    wire BUS_IN.O -> LUT6_A.I0 {
        thickness: 2;
        crossline: 8; // Indicates an 8-bit bus
    }

    // Optional Layout
    layout {
        LUT6_A at (10, 10);
        DFF_1 at (100, 10);
    }
}
```

## 4. Layout and Rendering

- **Logical First**: Connectivity is defined before layout.
- **Automatic Layout**: Default grid-based or flow-based layout (Phase 2).
- **Manual Refinement**: Deferred coordinate assignment via `layout` block.
- **Styles**: Support for multiple rendering styles (e.g., "Xilinx", "Generic").

## 5. Roadmap

### Phase 1: Core Language and Renderer
- Define core DSL for blocks and wires.
- Basic `draw2d` rendering engine (rectangles, lines, text).
- Support for basic logic gates (`AND`, `OR`, `NOT`).
- Basic parameter and pin specification.

### Phase 2: Advanced Features
- Implementation of the full IEEE 91-1984 shape library.
- Functional Dependency Notation support (`G`, `C`, `S`, `R`, `EN`).
- Complex MSI blocks (`REG`, `CTR`, `ALU`).
- Custom rendering functions for `CUSTOM` blocks.
- Basic auto-layout (grid layout).
- Improved wire routing (orthogonal).

### Phase 3: Tooling
- CLI tool for rendering.
- Library of common hardware blocks.


## 5. Rendering Engine and Implementation

- **Implementation Language**: Go (Golang).
- **Rendering Library**: [draw2d](https://github.com/llgcode/draw2d) for 2D vector graphics.
- **Output Formats**: Export to SVG, PDF, and PNG via `draw2d` or complementary libraries.
- **Routing**: Automatic orthogonal routing with minimal crossings.
- **Precision**: LaTeX-like precision for text placement and port alignment.

## 6. Project Structure and Task Management

- **Specification**: Stored in `//ai/spec.md`.
- **Tasks**: Stored in `//ai/tasks/`.
- **Naming Convention**: All task files must start with a `NNN` prefix (e.g., `000_initial_lexer.md`), where `NNN` is a sequence number starting from `000`.

lementary libraries.
- **Routing**: Automatic orthogonal routing with minimal crossings.
- **Precision**: LaTeX-like precision for text placement and port alignment.

## 6. Project Structure and Task Management

- **Specification**: Stored in `//ai/spec.md`.
- **Tasks**: Stored in `//ai/tasks/`.
- **Naming Convention**: All task files must start with a `NNN` prefix (e.g., `000_initial_lexer.md`), where `NNN` is a sequence number starting from `000`.

