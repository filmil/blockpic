#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Fails when pandoc could not find a figure for the book's PDF. pandoc only
# warns ("Could not fetch resource"), so the PDF builds without it.
set -euo pipefail
log=""
for f in "$@"; do
  case "$f" in *.log) log="$f" ;; esac
done
[ -n "$log" ] || { echo "no log among: $*" >&2; exit 1; }
if grep -n 'Could not fetch resource' "$log"; then
  echo "the book's PDF is missing the figures above" >&2
  exit 1
fi
echo "PASS: the book's PDF found every figure"
