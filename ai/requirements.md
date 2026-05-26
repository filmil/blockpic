# Hodos


## Synopsis

* A language for engineering block diagram typesetting, and a rendering engine
for it.
* The element to have esthetics similar to those of xilinx
ug474
* Implementation language is go.
  * You can use the draw2d library for rendering.

* No manual layout until last moment. Have a way to declare blocks first, then
  wires, and only then layout - maybe.


## Workspace


* Write spec files into dir `//ai`.
* Write tasks into dir `//ai/tasks`.
  * Any files must start with a NNN prefix, where NNN is a sequence number of
    the file, starting from `000` and incrementing for each next file.
