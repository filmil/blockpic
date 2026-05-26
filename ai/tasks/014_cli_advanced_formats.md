# Task: Advanced CLI Output Formats (Phase 3)

## Objective
Extend the CLI tool to support SVG and PDF generation.

## Requirements
- Integrate `draw2dsvg` and `draw2dpdf` (or equivalents) into the renderer package.
- Update `hodos render` to accept and handle `--format svg` and `--format pdf`.
- Ensure multi-format output correctly scales vector graphics.

## Deliverables
- Updated `main` and `renderer` packages.
- Tests verifying the creation of valid SVG/PDF files.
