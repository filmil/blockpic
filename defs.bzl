load("@npm//:netlistsvg/package_json.bzl", "bin")

_DEFAULT_SKIN = "//ai:skin_xilinx.svg"

def blockpic_diagram(name, src, out, skin = _DEFAULT_SKIN, **kwargs):
    """
    Renders a Blockpic diagram to SVG using netlistsvg.

    Args:
      name: The name of the target that gives `out`.
      src: The `.blockpic` file to draw.
      out: The SVG file to write.
      skin: The netlistsvg skin to draw with. The default,
        `//ai:skin_xilinx.svg`, is in the style of AMD's UG474 figures.
        `//ai:skin_with_arrows.svg` is the earlier look: a bold monospace
        face, and arrowheads on the nets.
      **kwargs: Passed to the target that gives `out`, such as
        `visibility` or `tags`.
    """
    json_out = name + ".json"

    native.genrule(
        name = name + "_json",
        srcs = [src],
        outs = [json_out],
        cmd = "$(location //cmd/blockpic) render $< -format yosys-json -o $@",
        tools = ["//cmd/blockpic"],
    )

    bin.netlistsvg_binary(
        name = name + "_tool",
    )
    intermediate_svg = name + "_raw.svg"
    native.genrule(
        name = name + "_raw",
        srcs = [json_out, skin],
        outs = [intermediate_svg],
        cmd = "BAZEL_BINDIR=. $(location :" + name + "_tool) $(location " + json_out + ") --skin $(location " + skin + ") -o $@",
        tools = [":" + name + "_tool"],
    )

    native.genrule(
        name = name,
        srcs = [src, intermediate_svg],
        outs = [out],
        cmd = "$(location //cmd/blockpic) refine $(location " + src + ") -svg $(location " + intermediate_svg + ") -o $@",
        tools = ["//cmd/blockpic"],
        **kwargs
    )

def blockpic_netlist_diagram(name, netlist, out, top = "", skin = _DEFAULT_SKIN, **kwargs):
    """Draws a Yosys JSON netlist as an SVG.

    The netlist is imported with `blockpic import` into
    `<name>.blockpic`, which is then drawn as `blockpic_diagram` draws a
    hand-written one. A `vivado_synthesis` target of rules_openxc7 gives
    such a netlist:

    ```python
    vivado_synthesis(name = "counter_synth", project = ":counter")

    blockpic_netlist_diagram(
        name = "counter_diagram",
        netlist = ":counter_synth",
        out = "counter.svg",
    )
    ```

    Args:
      name: The name of the target that gives `out`. `<name>_src` gives
        the imported `<name>.blockpic`.
      netlist: A label whose only output is a Yosys JSON netlist, such as
        a `vivado_synthesis` target.
      out: The SVG file to write.
      top: The module to draw. When empty, the module that Yosys marked
        as the top is drawn.
      skin: The netlistsvg skin to draw with, as for `blockpic_diagram`.
      **kwargs: Passed to the target that gives `out`, such as
        `visibility` or `tags`.
    """
    src = name + ".blockpic"
    top_flag = " -top " + top if top else ""
    native.genrule(
        name = name + "_src",
        srcs = [netlist],
        outs = [src],
        cmd = "$(location //cmd/blockpic) import $(location " + netlist + ")" + top_flag + " -o $@",
        tools = ["//cmd/blockpic"],
        visibility = kwargs.get("visibility"),
    )
    blockpic_diagram(name = name, src = src, out = out, skin = skin, **kwargs)
