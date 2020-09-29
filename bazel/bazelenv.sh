#!/bin/bash

BASE_DIR="$(mktemp -d)"

export USE_BAZEL=1
export BAZEL_PATH="/tmp/testBazel"
export BAZEL_HOME="$BASE_DIR/bazelhome"
export BAZEL_OUTPUT_BASE="$BASE_DIR/output"
export BAZEL_WORKSPACE="$(pwd)"

mkdir -p $BAZEL_HOME
mkdir -p $BAZEL_OUTPUT_BASE
