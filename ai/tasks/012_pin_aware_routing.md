# Task: Pin-aware Orthogonal Routing (Phase 2)

## Objective
Enhance the router to use specific pin positions for start and end points of wires.

## Requirements
- Calculate (x, y) coordinates for pins based on their position hint (left, right, top, bottom) on the block boundary.
- Update `Route` function to take pin-specific start/end points.
- Ensure wires exit/enter blocks perpendicular to the boundary.

## Deliverables
- Updated `router` and `renderer` packages.
- Visual test cases with complex pin placements.
