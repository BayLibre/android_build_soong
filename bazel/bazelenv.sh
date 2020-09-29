#!/bin/bash

# Helper script for setting environment variables required for Bazel/Soong
# mixed builds prototype. For development use only.
#
# Usage:
#   bazelenv.sh [absolute_bazel_path]
#
# If absolute_bazel_path is not specified, `which bazel` will be used
# to locate the appropriate bazel to use.


# Function to find top of the source tree (if $TOP isn't set) by walking up the
# tree.
function gettop
{
    local TOPFILE=build/soong/root.bp
    if [ -n "${TOP-}" -a -f "${TOP-}/${TOPFILE}" ] ; then
        # The following circumlocution ensures we remove symlinks from TOP.
        (cd $TOP; PWD= /bin/pwd)
    else
        if [ -f $TOPFILE ] ; then
            # The following circumlocution (repeated below as well) ensures
            # that we record the true directory name and not one that is
            # faked up with symlink names.
            PWD= /bin/pwd
        else
            local HERE=$PWD
            T=
            while [ \( ! \( -f $TOPFILE \) \) -a \( $PWD != "/" \) ]; do
                \cd ..
                T=`PWD= /bin/pwd -P`
            done
            \cd $HERE
            if [ -f "$T/$TOPFILE" ]; then
                echo $T
            fi
        fi
    fi
}

BASE_DIR="$(mktemp -d)"

if [[ $# -eq 0 ]] ; then
    export BAZEL_PATH="$(which bazel)"
else
    export BAZEL_PATH="$1"
fi

export USE_BAZEL=1
export BAZEL_HOME="$BASE_DIR/bazelhome"
export BAZEL_OUTPUT_BASE="$BASE_DIR/output"
export BAZEL_WORKSPACE="$(gettop)"

mkdir -p $BAZEL_HOME
mkdir -p $BAZEL_OUTPUT_BASE
