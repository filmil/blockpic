#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Checks that blockpic drew the diagram: an SVG that names its blocks.
set -euo pipefail
svg="$1"
grep -q '<svg' "$svg" || { echo "not an SVG: $svg" >&2; exit 1; }
for block in in_a in_b sum led; do
  grep -q "$block" "$svg" || { echo "block $block missing from $svg" >&2; exit 1; }
done
echo "PASS: $svg draws in_a, in_b, sum and led"
