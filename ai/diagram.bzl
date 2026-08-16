load("//:defs.bzl", "blockpic_diagram")

def render_diagram(name, src):
    blockpic_diagram(
        name = name,
        src = src,
        out = name + "_gen.svg",
    )
