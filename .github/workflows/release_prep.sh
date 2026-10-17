#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Builds the release's source archive and prints the release notes to stdout.
#
# Called by bazel-contrib/.github/.github/workflows/release_ruleset.yaml,
# which requires this exact path, runs it as
# `release_prep.sh TAG > release_notes.txt`, and attests the archive.
set -o errexit -o nounset -o pipefail

TAG="$1"
VERSION="${TAG#v}"
ARCHIVE="blockpic-${TAG}.zip"

# --symlinks: store a symlink as a symlink. Without it zip follows Bazel's
# convenience symlinks (bazel-*, also integration/bazel-*) into the output
# base, and it walks before -x filters, so the exclusion alone does not
# stop that.
# release_notes.txt: the shell creates it before this script starts.
zip --quiet --symlinks --recurse-paths "${ARCHIVE}" . \
  -x '*.git*' '*bazel-*' '*node_modules/*' 'user.bazelrc' \
     'release_notes.txt' "${ARCHIVE}"

cat <<NOTES
## Using Bzlmod

\`\`\`starlark
bazel_dep(name = "blockpic", version = "${VERSION}")
\`\`\`

Then, in a \`BUILD.bazel\` file:

\`\`\`starlark
load("@blockpic//:defs.bzl", "blockpic_diagram")
\`\`\`
NOTES
