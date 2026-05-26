# Task: Basic Rendering Engine (Phase 1)

## Objective
Implement a basic rendering engine using `draw2d` to visualize the logical structure.

## Requirements
- Initialize `draw2d` context for PNG and SVG output.
- Implement basic shape rendering: `Rectangle` (for blocks), `Line` (for wires).
- Support text rendering for block names and pin labels.
- Implement standard UG474 color palette and line weights.

## Deliverables
- `renderer` package in Go.
- Integration test that takes a parsed AST and produces a simple PNG.
