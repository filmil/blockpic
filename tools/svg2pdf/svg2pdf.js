// SPDX-License-Identifier: Apache-2.0
//
// Converts an SVG into a one-page PDF of the same size, so that pdflatex
// can \includegraphics a blockpic drawing.
//
//   svg2pdf <in.svg> <out.pdf>
//
// The page is the SVG's own width and height, a pixel taken as a point.
// All text is set in DejaVu Sans Mono, embedded, since blockpic's skins
// draw only in a monospace face. Exits with status 1 on a bad argument or
// an unreadable file.
"use strict";

const fs = require("fs");
const path = require("path");
const PDFDocument = require("pdfkit");
const SVGtoPDF = require("svg-to-pdfkit");

function main(argv) {
  if (argv.length !== 2) {
    console.error("usage: svg2pdf <in.svg> <out.pdf>");
    process.exit(1);
  }
  const [input, output] = argv;
  const svg = fs.readFileSync(input, "utf8");
  const root = svg.match(/<svg\b[^>]*>/);
  const size = (name) => {
    const m = root && root[0].match(new RegExp(`\\b${name}="([0-9.]+)"`));
    return m ? Number(m[1]) : NaN;
  };
  const width = size("width");
  const height = size("height");
  if (!(width > 0 && height > 0)) {
    console.error(`svg2pdf: ${input} has no numeric width and height`);
    process.exit(1);
  }

  // A fixed date keeps the PDF the same from one build to the next.
  const epoch = new Date(0);
  const doc = new PDFDocument({
    size: [width, height],
    margin: 0,
    info: { CreationDate: epoch, ModDate: epoch, Producer: "blockpic svg2pdf" },
  });
  const fonts = path.join(path.dirname(require.resolve("dejavu-fonts-ttf/package.json")), "ttf");
  doc.registerFont("mono", path.join(fonts, "DejaVuSansMono.ttf"));
  doc.registerFont("mono-bold", path.join(fonts, "DejaVuSansMono-Bold.ttf"));
  const out = fs.createWriteStream(output);
  doc.pipe(out);
  SVGtoPDF(doc, svg, 0, 0, {
    width,
    height,
    assumePt: true,
    fontCallback: (family, bold) => (bold ? "mono-bold" : "mono"),
  });
  doc.end();
  out.on("finish", () => process.exit(0));
}

main(process.argv.slice(2));
