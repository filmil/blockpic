# Blockpic


## Synopsis

* A language for engineering block diagram typesetting, and a rendering engine
for it.
* The element to have esthetics similar to those of xilinx
ug474
* Implementation language is go.
  * You can use the draw2d library for rendering.

* No manual layout until last moment. Have a way to declare blocks first, then
  wires, and only then layout - maybe.

* For each block, you can specify:
  * Name
  * Input pins
  * Output pins
  * Parameters (key-value pairs)
  * A "kind" (e.g., "AND", "OR", "FLIPFLOP", "CUSTOM")
  * Dimensions (width, height) - maybe.
  * A rendering function (for custom blocks) - maybe.

* For each wire, you can specify:
  * Source pin (block name, pin name)
  * Destination pin (block name, pin name)
  * Routing hints (e.g., "straight", "L-shape", "Z-shape") - maybe.
  * Bus thickness.
  * Bus crossline.

* The rendering engine should:
  * Take a block diagram description as input.
  * Render it to an image (e.g., PNG, SVG).
  * Support different rendering styles (e.g., "Xilinx", "Generic").
  * Allow for custom block rendering.

* The language should be simple and easy to read/write.
  * Potentially a YAML or JSON-based format.
  * Or a custom DSL.

* Add shape library that consists of digital blocks, IEEE standards based
  shaping:
  - AND
  - OR
  - NOT
  - XOR
  - FLIPFLOP (D, JK, T, SR)
  - MUX (2-to-1, 4-to-1, etc.)
  - DEMUX
  - REGISTER
  - COUNTER
  - SHIFT REGISTER
  - CLOCK
  - CONSTANT (HIGH, LOW)
  - INPUT PIN
  - OUTPUT PIN
  - TRI-STATE BUFFER
  - PULL-UP/PULL-DOWN RESISTOR
  - VCC/GND symbols
  - Custom block (user-defined shape and pins)

* See standards document for how to render blocks: https://wakerly.org/DDPP/DDPP3_pdf/IEEEsyms.pdf

## Goals

* **Phase 1: Core Language and Renderer**
  * Define the core language for blocks and wires.
  * Implement a basic rendering engine using `draw2d`.
  * Support basic shapes (rectangles, lines, text).
  * Implement a few standard block kinds (AND, OR, NOT).
  * Allow for basic parameter specification.

* **Phase 2: Advanced Features**
  * Implement more complex block kinds (FLIPFLOP, MUX, etc.).
  * Add support for custom block rendering functions.
  * Implement basic auto-layout algorithms (e.g., simple grid layout).
  * Add support for different rendering styles.
  * Improve wire routing algorithms.

* **Phase 3: Tooling and Ecosystem**
  * Develop a CLI tool for compiling and rendering diagrams.
  * Consider integration with other tools (e.g., VS Code extension).
  * Create a library of common blocks.
  * We must have a documentation section which shows (1) the original xilinx layout, (2) the layout converted to blockpic and (3) the blockpic source, all three side by side.


## Workspace


* Write spec files into dir `//ai`.
* Write tasks into dir `//ai/tasks`.
  * Any files must start with a NNN prefix, where NNN is a sequence number of
    the file, starting from `000` and incrementing for each next file.
