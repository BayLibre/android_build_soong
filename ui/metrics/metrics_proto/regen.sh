#!/bin/bash

# Generates the golang source file of metrics.proto protobuf file.

set -e

function die() { echo "ERROR: $1" >&2; exit 1; }

if ! hash aprotoc &>/dev/null; then
  die "could not find aprotoc. Maybe you need to run 'lunch aosp_arm-eng && m aprotoc blueprint_tools'?"
fi

if ! aprotoc --go_out=paths=source_relative:. metrics.proto; then
  die "build failed."
fi
