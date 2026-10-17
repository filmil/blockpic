#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Checks what blockpic drew for this module: SVGs that name what they draw.
set -euo pipefail
example="$1" arrows="$2" counter="$3"

expect() {  # file, then the words it must contain
  local svg="$1"; shift
  grep -q '<svg' "$svg" || { echo "not an SVG: $svg" >&2; exit 1; }
  for w in "$@"; do
    grep -q "$w" "$svg" || { echo "$w missing from $svg" >&2; exit 1; }
  done
  echo "PASS: $svg draws $*"
}

expect "$example" in_a in_b sum led
expect "$arrows" in_a in_b sum led
expect "$counter" FDRE CARRY4 IBUF OBUF
# The two skins must draw differently.
if cmp -s "$example" "$arrows"; then
  echo "the default and the arrows skin drew the same SVG" >&2; exit 1
fi
echo "PASS: the two skins differ"
