# Hodos

[![CI](https://github.com/filmil/hodos/actions/workflows/main.yml/badge.svg)](https://github.com/filmil/hodos/actions/workflows/main.yml)
[![Deploy Docs](https://github.com/filmil/hodos/actions/workflows/deploy_docs.yml/badge.svg)](https://github.com/filmil/hodos/actions/workflows/deploy_docs.yml)

Hodos is a language for engineering block diagram typesetting, designed to produce diagrams with the aesthetics of Xilinx UG474 technical documentation.

## Documentation

The project documentation is generated using Bazel and is available in PDF and HTML formats.

- **PDF**: `bazel build //ai:book_pdf`
- **HTML**: `bazel build //ai:html_chunked`

## Implementation

Implemented in Go using the `draw2d` library for vector graphics.
