# Task: Advanced Text Rendering (Phase 2)

## Objective
Implement robust text rendering within diagrams, ensuring clear readability, proper alignment, and support for various text elements.

## Requirements
- Replace the basic `FillStringAt` calls with a more sophisticated text rendering approach that handles bounding boxes.
- Implement text measuring to ensure block dimensions (if dynamically sized) can accommodate long block names, parameter lists, or complex pin labels.
- Support multi-line text for block parameters (e.g., `init: 0x8000\nmode: fast`).
- Ensure text does not overlap with wire routing or block boundaries.

## Deliverables
- Updated `renderer` package with robust text handling.
- Visual test cases demonstrating multi-line text and long labels within various block shapes.
