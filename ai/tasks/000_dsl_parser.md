# Task: DSL Parser and Lexer (Phase 1)

## Objective
Implement the initial lexer and parser for the Hodos DSL to support the logical declaration of blocks and wires.

## Requirements
- Support `diagram`, `block`, and `wire` top-level declarations.
- Parse block attributes: `kind`, `inputs`, `outputs`, `params`, `size`.
- Parse wire attributes: `source`, `destination`, `thickness`, `crossline`, `hint`.
- Implement basic error handling for syntax errors.

## Deliverables
- Lexer and Parser implementation in Go (using `gocc` or manual recursive descent).
- Unit tests for sample `.hodos` files.
