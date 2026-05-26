# Task: Pin and Bus Rendering Refinement (Phase 1/2)

## Objective
Refine the renderer to explicitly show pins and differentiate between single wires and multi-bit buses.

## Requirements
- Render pins as small attachment points or labels on the block boundary.
- Implement IEEE functional indicators:
    - **Bubble (○)**: External negation indicator.
    - **Wedge (>)**: Dynamic/Edge-triggered indicator.
    - **Triangle (▽)**: Three-state indicator.
- Implement visual differentiation for buses (thicker lines) as specified in `spec.md`.
- Support crossline indicators for buses with width labels.

## Deliverables
- Updated `renderer` package.
- Unit tests/Visual snapshots.
