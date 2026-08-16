load("@npm//:netlistsvg/package_json.bzl", "bin")

def blockpic_diagram(name, src, out):
    """
    Renders a Blockpic diagram to SVG using netlistsvg.
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
        srcs = [json_out, "//ai:skin_with_arrows.svg"],
        outs = [intermediate_svg],
        cmd = "BAZEL_BINDIR=. $(location :" + name + "_tool) $(location " + json_out + ") --skin $(location //ai:skin_with_arrows.svg) -o $@",
        tools = [":" + name + "_tool"],
    )

    native.genrule(
        name = name,
        srcs = [src, intermediate_svg],
        outs = [out],
        cmd = "$(location //cmd/blockpic) refine $(location " + src + ") -svg $(location " + intermediate_svg + ") -o $@",
        tools = ["//cmd/blockpic"],
    )
