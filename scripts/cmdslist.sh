#!/usr/bin/env bash
#
# makeall.sh generates binaries for everything under src/cmd/
set -eu

for dir in $(find src/cmd -mindepth 1 -maxdepth 1 -type d); do
  binname=${dir##*/}
  echo bin/$binname
done
