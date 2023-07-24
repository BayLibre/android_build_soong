#!/usr/bin/env bash

###############
# Removes the Bazel output base.
# This is intended to solve an issue when a build top is moved.
# Starlark symlinks are absolute and a moved build top will have many
# dangling symlinks and fail to function as intended.
###############

if [[ ! -v $OUT_DIR ]]; then
    dir_to_remove="out/bazel/output"
else
    dir_to_remove="$OUT_DIR/bazel/output"
fi

rm -rf $dir_to_remove
