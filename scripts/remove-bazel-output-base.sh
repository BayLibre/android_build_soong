#!/usr/bin/env bash

###############
# Removes the Bazel output base.
# This is intended to solve an issue when a build top is moved.
# Starlark symlinks are absolute and a moved build top will have many
# dangling symlinks and fail to function as intended.
# You MUST lunch again after moving your build top, before running this.
###############

if [[ ! -v $OUT_DIR ]]; then
    dir_to_remove="$ANDROID_BUILD_TOP/out/bazel/output"
else
    dir_to_remove="$ANDROID_BUILD_TOP/$OUT_DIR/bazel/output"
fi

read -p "Are you sure you want to remove $dir_to_remove? Y/N " -n 1 -r
echo    # (optional) move to a new line
if [[ $REPLY =~ ^[Yy]$ ]]
then
   rm -rf $dir_to_remove
else
   echo "Run lunch in your new checkout and try again"
fi
