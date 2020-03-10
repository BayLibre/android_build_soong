#!/bin/bash

set -e

function die() { echo "ERROR: $1" >&2; exit 1; }

readonly build_error_msg="Maybe you need to run 'lunch aosp_arm-eng && m aprotoc blueprint_tools'?"

if ! hash aprotoc &>/dev/null; then
  die "could not find aprotoc. ${build_error_msg}"
fi

if ! aprotoc --go_out=paths=source_relative:. build_completion.proto; then
  die "build failed. ${build_error_msg}"
fi
