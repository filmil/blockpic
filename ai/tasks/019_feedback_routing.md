# Task: Feedback Routing Signals (Phase 2/3)

## Objective
Implement routing capabilities for feedback signals, typical in state machines (Mealy/Moore) or sequential logic with loops.

## Requirements
- Update the `router` package to handle the `feedback` routing hint.
- Implement pathfinding that allows a wire to exit an output pin, route visually around the source and intermediate blocks (typically over or under them), and enter an input pin located horizontally "before" the source block.
- Ensure feedback loops do not intersect blocks unnecessarily and maintain orthogonal aesthetics.
- Coordinate with the layout engine (if auto-layout is used) to reserve routing channels above or below blocks for these feedback paths.

## Deliverables
- Updated `router` and `layout` packages.
- Visual test cases demonstrating state machine diagrams with complex feedback loops.
