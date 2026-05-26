def render_diagram(name, src):
    native.genrule(
        name = "gen_" + name + "_example",
        srcs = [src],
        outs = [name + "_gen.png"],
        cmd = "$(location //cmd/hodos) render $(location " + src + ") -o $@",
        tools = ["//cmd/hodos"],
    )
