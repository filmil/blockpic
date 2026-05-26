# Task: Refine IEEE Graphics to Exact Standards (Phase 2)

## Objective
Redo the element graphics to match the IEEE standard (ANSI/IEEE Std 91-1984) exactly as shown in the provided PDF document (https://wakerly.org/DDPP/DDPP3_pdf/IEEEsyms.pdf).

## Requirements
- Review the current `shapes` package implementations for basic gates (AND, OR, NOT, XOR) and complex MSI blocks.
- Update the drawing algorithms to perfectly replicate the proportions, curve radii, and relative sizing shown in the official documentation.
- Specifically ensure that the rectangular notation for MSI blocks correctly sizes the "Common Control Block" and visually distinguishes it according to the standard.
- Ensure functional indicators (bubbles, wedges, triangles) are sized and positioned perfectly on the block boundaries.

## Deliverables
- Updated `shapes` and `renderer` packages.
- Regenerated `shapes.svg`, `msi_blocks.svg`, and `dynamic_sizing.svg` demonstrating the refined, exact IEEE standards.
