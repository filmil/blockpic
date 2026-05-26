# Hodos Datasheet: Implemented Features

This document provides a hierarchical overview of the features implemented in the Hodos DSL and rendering engine.

## 1. Core Language (DSL)
*   **AST Structure**: Support for `Diagram`, `Block`, `Wire`, and `Layout` definitions.
*   **Lexer/Parser**: Recursive descent parser capable of reading `.hodos` files.
    *   Block Attributes: `kind`, `inputs`, `outputs`, `params`, `size`.
    *   Wire Attributes: `source`, `destination`, `thickness`, `crossline`, `hint`.
    *   Layout Attributes: Explicit coordinate mapping (`at (x,y)`).

## 2. Rendering Engine
*   **Base Graphics**: Integration with `draw2d` for vector rendering.
*   **Aesthetics**: Default Xilinx UG474 styling (Light Blue blocks, Dark Grey outlines).
*   **Shapes Library**:
    *   Basic Logic: `AND`, `OR`, `NOT` (IEEE standard geometry).
    *   Sequential: `DFF` (D-Flip-Flop with clock tick).
    *   Routing: `MUX` (Trapezoidal multiplexer).
    *   Fallback: Standard rectangular block rendering.
*   **Wire Routing**:
    *   Support for `thickness` styling.
    *   Orthogonal routing paths (`straight`, `L-shape`, `Z-shape`).
    *   Bus width indicators (`crossline`).

## 3. Layout Engine
*   **Auto-Layout**: Basic flow-based coordinate assignment for unmapped blocks.

## 4. Tooling
*   **CLI Application**: `hodos` binary.
    *   `hodos compile <file>`: Validates DSL syntax.
    *   `hodos render <file> [-format png] [-o output]`: Generates diagram images.

## 5. Continuous Delivery
*   **Automated Releases**: GitHub Workflow for multi-arch binary releases (Linux, macOS).
*   **Versioning**: Automatic semantic version tagging.

## 6. IEEE Standard Shape Library
Hodos provides a comprehensive library of digital logic blocks based on IEEE standards.

### 6.1 Supported Shapes
*   **Logic Gates**: `AND`, `OR`, `NOT`, `XOR`.
*   **Sequential Logic**: `DFF`, `JKFF`, `TFF`, `SRFF`.
*   **Routing**: `MUX`, `DEMUX`.
*   **Power/Signal**: `VCC`, `GND`, `CLOCK`.

### 6.2 Visual Reference and Source Code
<table>
<tr>
<td valign="top" width="40%">

```hodos
diagram Shapes {
    block AND1 { kind: "AND"; inputs: A, B; outputs: O; }
    block OR1 { kind: "OR"; inputs: A, B; outputs: O; }
    block NOT1 { kind: "NOT"; inputs: I; outputs: O; }
    block XOR1 { kind: "XOR"; inputs: A, B; outputs: O; }
    block DFF1 { kind: "DFF"; inputs: D, CLK; outputs: Q; }
    block MUX1 { kind: "MUX"; inputs: I0, I1, S; outputs: O; }
    block DEMUX1 { kind: "DEMUX"; inputs: I, S; outputs: O0, O1; }
    block VCC1 { kind: "VCC"; outputs: V; }
    block GND1 { kind: "GND"; outputs: G; }
    block CLK1 { kind: "CLOCK"; outputs: C; }

    layout {
        AND1 at (20, 20)
        OR1 at (120, 20)
        NOT1 at (220, 20)
        XOR1 at (320, 20)
        DFF1 at (20, 100)
        MUX1 at (120, 100)
        DEMUX1 at (220, 100)
        VCC1 at (320, 100)
        GND1 at (20, 180)
        CLK1 at (120, 180)
    }
}
```

</td>
<td valign="top" width="60%">

![IEEE Shapes](ai/shapes.svg)

</td>
</tr>
</table>

## 7. AST Validation
Hodos includes a validation layer to ensure logical correctness before rendering.
*   **Unique Names**: Blocks must have unique identifiers.
*   **Connectivity**: Wire source and destination blocks must exist.
*   **Pin Verification**: Referenced pins must exist on their respective blocks.

## 8. Pins and Buses
Hodos supports explicit pin rendering and multi-bit bus visualization.
*   **Pin Rendering**: Named inputs and outputs are rendered on block boundaries.
*   **Bus Thickness**: Wires can have custom thickness to represent buses.
*   **Crossline Indicator**: Standard bus width indicators with labels.

### 8.1 Visual Reference and Source Code
<table>
<tr>
<td valign="top" width="40%">

```hodos
diagram PinsAndBuses {
    block REG1 {
        kind: "REGISTER";
        inputs: D, CLK, EN;
        outputs: Q;
    }

    block MUX1 {
        kind: "MUX";
        inputs: I0, I1, S;
        outputs: O;
    }

    wire REG1.Q -> MUX1.I0 {
        thickness: 3;
        crossline: 32;
        hint: "Z-shape";
    }

    layout {
        REG1 at (20, 20)
        MUX1 at (150, 20)
    }
}
```

</td>
<td valign="top" width="60%">

![Pins and Buses](ai/pins_buses.svg)

</td>
</tr>
</table>

## 9. Dynamic Gate Sizing
Hodos automatically calculates block dimensions based on the number of input and output pins.
*   **Automatic Height**: Blocks expand vertically to accommodate multiple pins while maintaining even distribution.
*   **Precise Pin Positioning**: Pins are positioned exactly at the boundary intersection, ensuring clean orthogonal routing.

### 9.1 Visual Reference and Source Code
<table>
<tr>
<td valign="top" width="40%">

```hodos
diagram DynamicSizing {
    block LUT_MANY {
        kind: "LUT";
        inputs: I0, I1, I2, I3, I4, I5, I6, I7;
        outputs: O0, O1;
    }

    block SMALL {
        kind: "NOT";
        inputs: I;
        outputs: O;
    }

    wire LUT_MANY.O0 -> SMALL.I {
        hint: "Z-shape";
    }

    layout {
        LUT_MANY at (20, 20)
        SMALL at (200, 40)
    }
}
```

</td>
<td valign="top" width="60%">

![Dynamic Sizing](ai/dynamic_sizing.svg)

</td>
</tr>
</table>

## 10. Registers, Counters, and Dependency Notation
Hodos strictly adheres to IEEE Std 91-1984 for MSI components.
*   **Common Control Block**: MSI blocks like `REGISTER` and `COUNTER` feature a separated top section for common controls.
*   **Dependency Notation**: Pins are labeled using IEEE functional indicators (`C` for clock, `EN` for enable, `R` for reset, `S` for set).
*   **Dynamic Indicators**: Clocks (`CLK`, `C`) are rendered with the standard `>` wedge indicator.

### 10.1 Visual Reference and Source Code
<table>
<tr>
<td valign="top" width="40%">

```hodos
diagram MSIBlocks {
    block REG8 {
        kind: "REGISTER";
        inputs: D [left], CLK [left], EN [left], R [left];
        outputs: Q [right];
        params: { width: "8" };
    }

    block CTR4 {
        kind: "COUNTER";
        inputs: CLK [left], EN [left], R [left];
        outputs: Q [right], TC [right];
        params: { width: "4" };
    }

    layout {
        REG8 at (20, 20)
        CTR4 at (200, 20)
    }
}
```

</td>
<td valign="top" width="60%">

![MSI Blocks](ai/msi_blocks.svg)

</td>
</tr>
</table>

## 11. Pin-Aware Routing
Hodos routes wires orthoganally, considering the physical side of the block where the pin is located.
*   **Directional Exits**: Wires exit perpendicular to the boundary they originate from.
*   **Directional Entries**: Wires enter perpendicular to the boundary they terminate at.
*   **Supported Hints**: `straight`, `L-shape`, `Z-shape` paths intelligently adapt to top/bottom/left/right pin orientations.

### 11.1 Visual Reference and Source Code
<table>
<tr>
<td valign="top" width="40%">

```hodos
diagram PinAware {
    block SRC {
        kind: "CUSTOM";
        inputs: I_T [top], I_B [bottom], I_L [left];
        outputs: O_R [right], O_T [top], O_B [bottom];
        size: (60, 60);
    }

    block DST_TOP {
        kind: "CUSTOM";
        inputs: I_B [bottom];
    }
    
    block DST_BOT {
        kind: "CUSTOM";
        inputs: I_T [top];
    }

    block DST_RIGHT {
        kind: "CUSTOM";
        inputs: I_L [left];
    }

    wire SRC.O_T -> DST_TOP.I_B { hint: "L-shape" }
    wire SRC.O_B -> DST_BOT.I_T { hint: "L-shape" }
    wire SRC.O_R -> DST_RIGHT.I_L { hint: "straight" }

    layout {
        SRC at (100, 100)
        DST_TOP at (100, 20)
        DST_BOT at (100, 200)
        DST_RIGHT at (200, 100)
    }
}
```

</td>
<td valign="top" width="60%">

![Pin-Aware Routing](ai/pin_aware.svg)

</td>
</tr>
</table>


## 12. Advanced Auto-Layout
Hodos features a robust layer-based automatic layout engine.
*   **Dependency Grouping**: Blocks are grouped into columns (layers) based on their logical depth from inputs to outputs.
*   **Topological Sorting**: Longest-path layer assignment ensures clear left-to-right flow without manual coordinate mapping.
*   **Cycle Handling**: The algorithm gracefully handles feedback loops by assigning arbitrary starting layers to cyclic components.

### 12.1 Visual Reference and Source Code
<table>
<tr>
<td valign="top" width="40%">

```hodos
diagram ComplexLayout {
    block IN1 { kind: "INPUT_PIN"; outputs: O; }
    block IN2 { kind: "INPUT_PIN"; outputs: O; }
    
    block AND1 { kind: "AND"; inputs: I1, I2; outputs: O; }
    block OR1 { kind: "OR"; inputs: I1, I2; outputs: O; }
    block NOT1 { kind: "NOT"; inputs: I; outputs: O; }
    
    block OUT1 { kind: "OUTPUT_PIN"; inputs: I; }

    wire IN1.O -> AND1.I1
    wire IN2.O -> AND1.I2
    wire IN2.O -> OR1.I1
    wire AND1.O -> OR1.I2
    wire OR1.O -> NOT1.I
    wire NOT1.O -> OUT1.I

    // No layout block provided, forcing AutoLayout to run
}
```

</td>
<td valign="top" width="60%">

![Complex Auto Layout](ai/complex_layout.svg)

</td>
</tr>
</table>
