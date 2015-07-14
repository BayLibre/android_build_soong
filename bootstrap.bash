#!/bin/bash

set -e

ORIG_SRCDIR=$(dirname "${BASH_SOURCE[0]}")
BUILDDIR=""
if [[ "$ORIG_SRCDIR" != "." ]]; then
  if [[ ${SRCDIR:0:1} == '/' ]]; then
    BUILDDIR=$PWD
  else
    BUILDDIR=$(python -c "import os; print os.path.relpath('.', '$ORIG_SRCDIR')")
  fi
  cd $ORIG_SRCDIR
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

if [[ $BUILDDIR == "" ]]; then
  # Parse command line flags, but fail back to blueprint's bootstrap.bash
  NOCREATEFILES=0
  while getopts ":b:i:r" opt; do
    case $opt in
      b) BUILDDIR="$OPTARG";;
      i) NOCREATEFILES=1;;
      r) NOCREATEFILES=1;;
      \?) NOCREATEFILES=1;;
      :)
        echo "Option -$OPTARG requires an argument." >&2
        exit 1
        ;;
    esac
  done

  if [[ "$BUILDDIR" == "" ]]; then
    echo "FAILED: Must provide a build output directory"
    echo "  Either run bootstrap.bash from the output directory"
    echo "  or run from $$TOP and pass -b <output>"
    exit 1
  fi
else
  NOCREATEFILES=$#
fi

export BUILDDIR

if [[ $NOCREATEFILES -eq 0 ]]; then
    mkdir -p $BUILDDIR

    if [[ $(find $BUILDDIR -maxdepth 1 -name Android.bp) ]]; then
      echo "FAILED: The build directory must not be a source directory"
      exit 1
    fi

    if [[ ${BUILDDIR:0:1} == '/' ]]; then
      export SRCDIR_FROM_BUILDDIR=$(python -c "import os; print os.path.abspath('${SRCDIR}')")
    else
      export SRCDIR_FROM_BUILDDIR=$(python -c "import os; print os.path.relpath('.', '$BUILDDIR')")
    fi

    sed -e "s|@@BuildDir@@|${BUILDDIR}|" \
        -e "s|@@SrcDirFromBuildDir@@|${SRCDIR_FROM_BUILDDIR}|" \
        -e "s|@@PrebuiltOS@@|${PREBUILTOS}|" \
        "$SRCDIR/build/soong/soong.bootstrap.in" > $BUILDDIR/.soong.bootstrap
    ln -sf "${SRCDIR_FROM_BUILDDIR}/build/soong/soong.bash" $BUILDDIR/soong
fi

"$SRCDIR/build/blueprint/bootstrap.bash" "$@"
