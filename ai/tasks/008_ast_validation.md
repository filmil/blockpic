# Task: AST Validation Layer (Phase 1/2)

## Objective
Implement a validation layer to ensure the logical consistency of the parsed diagram.

## Requirements
- Check for duplicate block names.
- Verify that wire source and destination blocks exist.
- Verify that pins referenced in wires exist on the respective blocks.
- Check for circular dependencies (if applicable to rendering).

## Deliverables
- `validation` package in Go.
- Unit tests with invalid Hodos files.
