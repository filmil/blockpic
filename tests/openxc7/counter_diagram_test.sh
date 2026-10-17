#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Checks the drawing of the synthesised counter: its ports, one register
# per bit, and the buffers synth_xilinx puts on every pin.
set -euo pipefail

svg="$1"
src="$2"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

# Each of these is a block in the SVG, by its cell id.
for cell in clk reset led_0 led_1 led_2 led_3 IBUF_1 BUFG_1 CARRY4_1 \
    FDRE_1 FDRE_2 FDRE_3 FDRE_4 OBUF_1 OBUF_2 OBUF_3 OBUF_4; do
  grep -q "id=\"cell_${cell}\"" "$svg" || fail "no cell ${cell} in ${svg}"
done
[[ "$(grep -c 'kind: "FDRE"' "$src")" == 4 ]] || fail "want 4 FDRE in ${src}"
[[ "$(grep -c 'kind: "OBUF"' "$src")" == 4 ]] || fail "want 4 OBUF in ${src}"
grep -q 'name: "led\[3\]"' "$src" || fail "no led[3] in ${src}"
# The counter's carry chain: bit 1 of the sum is the D of the second bit.
grep -q 'wire CARRY4_1.O_1 -> FDRE_2.D' "$src" || fail "no CARRY4_1.O_1 -> FDRE_2.D in ${src}"
echo PASS
