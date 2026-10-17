# SPDX-License-Identifier: Apache-2.0
"""Helpers for the sample documents."""

def svg_to_pdf(name, src, out, **kwargs):
    """Converts a blockpic SVG into a PDF that pdflatex can include.

    Args:
      name: The name of the target that gives `out`.
      src: A label whose only output is the SVG, such as a
        `blockpic_diagram` target.
      out: The PDF file to write.
      **kwargs: Passed to the genrule, such as `visibility`.
    """
    native.genrule(
        name = name,
        srcs = [src],
        outs = [out],
        cmd = "BAZEL_BINDIR=. $(location //tools/svg2pdf) $(location " + src + ") $@",
        tools = ["//tools/svg2pdf"],
        **kwargs
    )
