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

- **Block**: The fundamental unit, typically a rectangle.
- **Mux**: A specialized trapezoidal block representing a multiplexer.
- **Flip-Flop**: A block specialized for sequential logic.
- **LUT**: A block representing a Look-Up Table.

### 2.2 Ports

Ports are attachment points for connections. They are located on the boundaries of blocks.

- **Position**: Ports can be positioned on the `left`, `right`, `top`, or
  `bottom` of a block.
- **Type**: Ports can be `input`, `output`, or `inout`.
- **Labels**: Ports have internal or external labels (e.g., `D`, `Q`, `CE`,
  `CLK`).

### 2.3 Connections (Wires and Buses)

Connections link ports together.

- **Orthogonal Routing**: All connections are routed using 90-degree bends.
- **Wire**: A standard signal line (1-bit).
- **Bus**: A thicker signal line representing multiple bits.
- **Bus Width**: Buses can have an optional width label (e.g., `[7:0]`).

### 2.4 Containers and Groups

- **Group**: A logical grouping of blocks and connections.
- **Container**: A physical or logical boundary (e.g., a "Slice" or "CLB")
  often rendered with a light grey background and dashed/dotted borders.

### 2.5 Aesthetics (UG474 Style)

- **Colors**:
    - Primitives: Light Blue (`#E6F2FF`) with a Dark Grey (`#333333`) outline.
    - Containers: Light Grey (`#F2F2F2`) or transparent with dashed borders.
    - Special Paths: Light Orange (`#FFF2E6`) for carry chains or shift registers.
- **Typography**: Sans-serif (Helvetica or Arial), 8pt to 10pt.
- **Line Weights**: 0.5pt for wires, 1.5pt for buses.

## 3. Syntax (Draft)

```hodos
diagram MyDiagram {
    config {
        font: "Helvetica", 9pt;
        default_color: #E6F2FF;
    }

    container SliceA {
        style: dashed;

        lut LUT6_A {
            pos: (10, 10);
            port in I0, I1, I2, I3, I4, I5 [left];
            port out O [right];
        }

        mux MUX7_A {
            pos: (100, 20);
            port in I0, I1 [left];
            port in S [bottom];
            port out O [right];
        }

        wire LUT6_A.O -> MUX7_A.I0;
    }
}
```

## 4. Layout Rules
- **Flow**: Diagrams generally flow from left to right.
- **Grid**: Elements are snapped to a coordinate grid for alignment.
- **Port Alignment**: Ports on opposite sides of a wire should align horizontally or vertically where possible to minimize bends.

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

