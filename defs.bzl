load("@npm//:netlistsvg/package_json.bzl", "bin")

def hodos_diagram(name, src, out):
    """
    Renders a Hodos diagram to SVG using netlistsvg.
    """
    json_out = name + ".json"
    
    native.genrule(
        name = name + "_json",
        srcs = [src],
        outs = [json_out],
        cmd = "$(location //cmd/hodos) render $< -format yosys-json -o $@",
        tools = ["//cmd/hodos"],
    )
    
    bin.netlistsvg_binary(
        name = name + "_tool",
    )
    native.genrule(
        name = name,
        srcs = [json_out],
        outs = [out],
        cmd = "BAZEL_BINDIR=. $(location :" + name + "_tool) $< -o $@",
        tools = [":" + name + "_tool"],
    )
