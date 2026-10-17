# Label(), not a string: in a macro a string label is read in the package
# that calls it, which for a module using blockpic is not this repository.
_BLOCKPIC = Label("//cmd/blockpic")
_NETLISTSVG = Label("//:netlistsvg")
_DEFAULT_SKIN = Label("//skins:skin_xilinx.svg")

def blockpic_diagram(name, src, out, skin = _DEFAULT_SKIN, **kwargs):
    """
    Renders a Blockpic diagram to SVG using netlistsvg.

    Args:
      name: The name of the target that gives `out`.
      src: The `.blockpic` file to draw.
      out: The SVG file to write.
      skin: The netlistsvg skin to draw with. The default,
        `@blockpic//skins:skin_xilinx.svg`, is in the style of AMD's UG474
        figures. `@blockpic//skins:skin_with_arrows.svg` is the earlier look:
        a bold monospace
        face, and arrowheads on the nets.
      **kwargs: Passed to the target that gives `out`, such as
        `visibility` or `tags`.
    """
    json_out = name + ".json"

    native.genrule(
        name = name + "_json",
        srcs = [src],
        outs = [json_out],
        cmd = "$(location {}) render $< -format yosys-json -o $@".format(_BLOCKPIC),
        tools = [_BLOCKPIC],
    )

    intermediate_svg = name + "_raw.svg"
    native.genrule(
        name = name + "_raw",
        srcs = [json_out, skin],
        outs = [intermediate_svg],
        cmd = "BAZEL_BINDIR=. $(location {}) $(location {}) --skin $(location {}) -o $@".format(
            _NETLISTSVG,
            json_out,
            skin,
        ),
        tools = [_NETLISTSVG],
    )

    native.genrule(
        name = name,
        srcs = [src, intermediate_svg],
        outs = [out],
        cmd = "$(location {}) refine $(location {}) -svg $(location {}) -o $@".format(
            _BLOCKPIC,
            src,
            intermediate_svg,
        ),
        tools = [_BLOCKPIC],
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
        cmd = "$(location {}) import $(location {}){} -o $@".format(_BLOCKPIC, netlist, top_flag),
        tools = [_BLOCKPIC],
        visibility = kwargs.get("visibility"),
    )
    blockpic_diagram(name = name, src = src, out = out, skin = skin, **kwargs)
