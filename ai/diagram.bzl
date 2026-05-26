load("//:defs.bzl", "hodos_diagram")

def render_diagram(name, src):
    hodos_diagram(
        name = name,
        src = src,
        out = name + "_gen.svg",
    )
