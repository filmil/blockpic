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
# --wild-stop-dirs: `*` stops at `/`, and `**` crosses it, so `bazel-*`
# means the top-level symlinks only, not a tracked docs/bazel-guide.md.
# Left out: git's files and the GitHub workflows, which the registry entry
# does not need; Bazel's symlinks; node_modules; the release notes, which
# the shell creates before this script starts.
zip --quiet --symlinks --wild-stop-dirs --recurse-paths "${ARCHIVE}" . \
  -x '.git' '.git/**' '.gitignore' '.github/**' \
     'bazel-*' 'integration/bazel-*' 'node_modules/**' 'user.bazelrc' \
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
