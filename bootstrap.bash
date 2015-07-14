#!/bin/bash

set -e

export BOOTSTRAP="./bootstrap.bash"
export SRCDIR="."
export BUILDDIR
export TOPNAME="Android.bp"
export BOOTSTRAP_MANIFEST="build/soong/build.ninja.in"
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
export GOROOT="prebuilts/go/$PREBUILTOS/"
export GOARCH="amd64"
export GOCHAR="6"

if [[ $(dirname "${BASH_SOURCE[0]}") != "." ]]; then
  echo "FAILED: bootstrap.bash must be run as './bootstrap.bash' from the source directory"
  exit 1
fi

# Parse command line flags, but fail back to blueprint's bootstrap.bash
CREATEFILES=1
BUILDDIR=""
while getopts ":b:i:" opt; do
  case $opt in
    b) BUILDDIR="$OPTARG";;
    i) CREATEFILES=0;;
    \?) CREATEFILES=0;;
    :)
      echo "Option -$OPTARG requires an argument." >&2
      exit 1
      ;;
  esac
done

if [[ "$BUILDDIR" == "" ]]; then
  echo "FAILED: Must provide a build output directory with -b <builddir>"
  exit 1
fi

if [[ $CREATEFILES -eq 1 ]]; then
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
        "build/soong/soong.bootstrap.in" > $BUILDDIR/.soong.bootstrap
    ln -sf "${SRCDIR_FROM_BUILDDIR}/build/soong/soong.bash" $BUILDDIR/soong
fi

"build/blueprint/bootstrap.bash" "$@"
