// SPDX-License-Identifier: Apache-2.0
//
// netlistsvg's three steps, one at a time, so that `blockpic layout` can
// work on the layout between them:
//
//   nsv graph <netlist.json> <skin.svg> <graph.json>
//       The ELK graph that netlistsvg would lay out, with the skin's
//       layout options on its root.
//   nsv layout <graph.json> <laid.json>
//       The graph laid out by ELK.
//   nsv draw <netlist.json> <skin.svg> <laid.json> <out.svg>
//       The netlist drawn with the given layout.
//
// `netlistsvg --layout` draws a given layout too, but in netlistsvg 1.0.2
// it resolves its promise without the drawing, so nothing is written.
// This tool calls the same functions that netlistsvg's own render does.
// Exits with status 1 on a bad argument or a failed step.
"use strict";

const fs = require("fs");
const path = require("path");

// netlistsvg's own dependencies resolve from its directory: elkjs, onml
// and json5 are its, at the versions it was built with.
const nsDir = path.dirname(require.resolve("netlistsvg/package.json"));
const own = (name) => require(require.resolve(name, { paths: [nsDir] }));
const ELK = own("elkjs");
const onml = own("onml");
const json5 = own("json5");
const Skin = require("netlistsvg/built/Skin").default;
const { FlatModule } = require("netlistsvg/built/FlatModule");
const { buildElkGraph } = require("netlistsvg/built/elkGraph");
const drawModule = require("netlistsvg/built/drawModule").default;

// flatModule loads the skin and the netlist as netlistsvg's render does,
// and builds the ELK graph. The graph also fills the lookup from edge to
// net that drawModule reads, so it is built before every drawing.
function flatModule(netlistPath, skinPath) {
  Skin.skin = onml.p(fs.readFileSync(skinPath, "utf8"));
  const props = Skin.getProperties();
  const fm = new FlatModule(json5.parse(fs.readFileSync(netlistPath, "utf8")));
  if (props.constants !== false) {
    fm.addConstants();
  }
  if (props.splitsAndJoins !== false) {
    fm.addSplitsJoins();
  }
  fm.createWires();
  const graph = buildElkGraph(fm);
  graph.layoutOptions = props.layoutEngine;
  return { fm, graph };
}

const commands = {
  graph(netlist, skin, out) {
    fs.writeFileSync(out, JSON.stringify(flatModule(netlist, skin).graph, null, 1));
  },
  layout(input, out) {
    return new ELK()
      .layout(JSON.parse(fs.readFileSync(input, "utf8")))
      .then((g) => fs.writeFileSync(out, JSON.stringify(g, null, 1)));
  },
  draw(netlist, skin, laid, out) {
    const { fm } = flatModule(netlist, skin);
    fs.writeFileSync(out, drawModule(JSON.parse(fs.readFileSync(laid, "utf8")), fm));
  },
};

function main(argv) {
  const run = commands[argv[0]];
  if (!run || argv.length !== run.length + 1) {
    console.error(
      "usage: nsv graph <netlist.json> <skin.svg> <graph.json>\n" +
        "       nsv layout <graph.json> <laid.json>\n" +
        "       nsv draw <netlist.json> <skin.svg> <laid.json> <out.svg>",
    );
    process.exit(1);
  }
  Promise.resolve()
    .then(() => run(...argv.slice(1)))
    .catch((e) => {
      console.error(`nsv ${argv[0]}: ${e.stack || e}`);
      process.exit(1);
    });
}

main(process.argv.slice(2));
