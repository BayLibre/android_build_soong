#!/bin/bash -eu

###############
# Removes the Bazel output base.
# This is intended to solve an issue when a build top is moved.
# Starlark symlinks are absolute and a moved build top will have many
# dangling symlinks and fail to function as intended.
# You MUST lunch again after moving your build top, before running this.
###############

if [[ ! -v ANDROID_BUILD_TOP ]]; then
    echo "ANDROID_BUILD_TOP not found in environment. Please run lunch before running this script"
    exit 1
fi

if [[ ! -v OUT_DIR ]]; then
    dir_to_remove="$ANDROID_BUILD_TOP/out/bazel/output"
else
    dir_to_remove="$ANDROID_BUILD_TOP/$OUT_DIR/bazel/output"
fi

if [[ ! -d $dir_to_remove ]]; then
    echo "The specified output directory doesn't exist."
    echo "Have you rerun lunch since moving directories?"
    exit 1
fi


read -p "Are you sure you want to remove $dir_to_remove? Y/N " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
   rm -rf $dir_to_remove
fi
