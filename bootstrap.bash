#!/bin/bash

set -e

ORIG_SRCDIR=$(dirname "${BASH_SOURCE[0]}")
if [[ "$ORIG_SRCDIR" != "." ]]; then
  if [[ ! -z "$BUILDDIR" ]]; then
    echo "error: To use BUILDDIR, run from the source directory"
    exit 1
  fi
  if [[ ${ORIG_SRCDIR:0:1} == '/' ]]; then
    export BUILDDIR=$PWD
  else
    export BUILDDIR=$(python -c "import os; print os.path.relpath('.', '$ORIG_SRCDIR')")
  fi
  cd $ORIG_SRCDIR
fi
if [[ -z "$BUILDDIR" ]]; then
  echo "error: Run ${BASH_SOURCE[0]} from the build output directory"
  exit 1
fi
export SRCDIR="."
export BOOTSTRAP="${SRCDIR}/bootstrap.bash"

export TOPNAME="Android.bp"
export BOOTSTRAP_MANIFEST="${SRCDIR}/build/soong/build.ninja.in"
export RUN_TESTS="-t"

case $(uname) in
    Linux)
	export GOOS="linux"
	export PREBUILTOS="linux-x86"
	;;
    Darwin)
	export GOOS="darwin"
	export PREBUILTOS="darwin-x86"
	;;
    *) echo "unknown OS:" $(uname) && exit 1;;
esac
export GOROOT="${SRCDIR}/prebuilts/go/$PREBUILTOS/"
export GOARCH="amd64"
export GOCHAR="6"

if [[ $# -eq 0 ]]; then
    mkdir -p $BUILDDIR

    if [[ $(find $BUILDDIR -maxdepth 1 -name Android.bp) ]]; then
      echo "FAILED: The build directory must not be a source directory"
      exit 1
    fi

    if [[ ${BUILDDIR:0:1} == '/' ]]; then
      export SRCDIR_FROM_BUILDDIR=$PWD
    else
      # We'd like to use relative paths here so that the source and build
      # directories can be moved around without rebuilding as long as they stay
      # in the same relative position. But relative paths don't always work when
      # there are symlinks involved:
      #
      # If BUILDDIR is a symlink to another directory in the same parent
      # directory (out -> out.angler), then using out and .. as relative paths
      # to get back and forth work fine.
      #
      # But if BUILDDIR is a symlink to another directory altogher (out ->
      # /mnt/ssd/out.master), then we shouldn't be relying on relative paths (so
      # that the source directory can still be moved).
      export SRCDIR_FROM_BUILDDIR=$(python -c "import os
realpath_relpath = os.path.relpath(os.path.realpath('.'), os.path.realpath('$BUILDDIR'))
relpath = os.path.relpath('.', '$BUILDDIR')
if realpath_relpath != relpath:
  print os.path.abspath('.')
else:
  print relpath")
    fi

    sed -e "s|@@BuildDir@@|${BUILDDIR}|" \
        -e "s|@@SrcDirFromBuildDir@@|${SRCDIR_FROM_BUILDDIR}|" \
        -e "s|@@PrebuiltOS@@|${PREBUILTOS}|" \
        "$SRCDIR/build/soong/soong.bootstrap.in" > $BUILDDIR/.soong.bootstrap
    ln -sf "${SRCDIR_FROM_BUILDDIR}/build/soong/soong.bash" $BUILDDIR/soong
fi

"$SRCDIR/build/blueprint/bootstrap.bash" "$@"
