# Task: Arrow Rendering for Signal Directionality (Phase 2)

## Objective
Implement arrowhead rendering on wires to explicitly indicate the direction of signal flow.

## Requirements
- Modify the `renderer` and `router` packages to calculate and draw an arrowhead at the terminal end of every wire (at the destination pin).
- The arrowhead should be oriented correctly based on the entry angle of the wire (e.g., pointing right if entering a left-side pin).
- Ensure the arrowhead aesthetics match the clean, minimalistic style of the existing lines.

## Deliverables
- Updated `renderer` package.
- Visual test case demonstrating correct arrow orientation on all four sides of a block.
