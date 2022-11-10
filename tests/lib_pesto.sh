#!/bin/bash -eu

set -o pipefail

HARDWIRED_MOCK_TOP=
# Uncomment this to be able to view the source tree after a test is run
# HARDWIRED_MOCK_TOP=/tmp/td

REAL_TOP="$(readlink -f "$(dirname "$0")"/../../..)"

if [[ -n "$HARDWIRED_MOCK_TOP" ]]; then
  MOCK_TOP="$HARDWIRED_MOCK_TOP"
else
  MOCK_TOP=$(mktemp -t -d st.XXXXX)
  trap cleanup_mock_top EXIT
fi

WARMED_UP_MOCK_TOP=$(mktemp -t soong_integration_tests_warmup.XXXXXX.tar.gz)
trap 'rm -f "$WARMED_UP_MOCK_TOP"' EXIT

function warmup_mock_top {
  info "Warming up mock top ..."
  info "Mock top warmup archive: $WARMED_UP_MOCK_TOP"
  cleanup_mock_top
  mkdir -p "$MOCK_TOP"
  cd "$MOCK_TOP"

  create_mock_make
  run_soong
  tar czf "$WARMED_UP_MOCK_TOP" *
}

function cleanup_mock_top {
  cd /
  rm -fr "$MOCK_TOP"
}

function info {
  echo -e "\e[92;1m[TEST HARNESS INFO]\e[0m" "$*"
}

function fail {
  echo -e "\e[91;1mFAILED:\e[0m" "$*"
  exit 1
}

function copy_directory {
  local dir="$1"
  local -r parent="$(dirname "$dir")"

  mkdir -p "$MOCK_TOP/$parent"
  cp -R "$REAL_TOP/$dir" "$MOCK_TOP/$parent"
}

function symlink_file {
  local file="$1"

  mkdir -p "$MOCK_TOP/$(dirname "$file")"
  ln -s "$REAL_TOP/$file" "$MOCK_TOP/$file"
}

function symlink_directory {
  local dir="$1"

  mkdir -p "$MOCK_TOP/$dir"
  # We need to symlink the contents of the directory individually instead of
  # using one symlink for the whole directory because finder.go doesn't follow
  # symlinks when looking for Android.bp files
  for i in "$REAL_TOP/$dir"/*; do
    i=$(basename "$i")
    local target="$MOCK_TOP/$dir/$i"
    local source="$REAL_TOP/$dir/$i"

    if [[ -e "$target" ]]; then
      if [[ ! -d "$source" || ! -d "$target" ]]; then
        fail "Trying to symlink $dir twice"
      fi
    else
      ln -s "$REAL_TOP/$dir/$i" "$MOCK_TOP/$dir/$i";
    fi
  done
}

function create_mock_soong {
  create_mock_bazel
  copy_directory build/blueprint
  copy_directory build/soong
  copy_directory build/make/tools/rbcrun

  symlink_directory prebuilts/sdk
  symlink_directory prebuilts/go
  symlink_directory prebuilts/build-tools
  symlink_directory prebuilts/clang/host
  symlink_directory external/go-cmp
  symlink_directory external/golang-protobuf
  symlink_directory external/starlark-go

  touch "$MOCK_TOP/Android.bp"
}

function create_mock_make {
  create_mock_soong
  
  # Generate make environment
  copy_directory build/core
  copy_directory build/make
  remove_file build/make/target/board/Android.mk
  remove_dir build/make/target/product/gsi
  remove_dir build/make/target/product/sysconfig
  remove_dir build/make/tools
  copy_directory build/make/tools/rbcrun
  copy_directory build/make/tools/releasetools
  copy_directory build/target
  remove_dir build/target/product/security
  copy_directory build/tools
  remove_dir build/tools/rbcrun
  remove_dir build/tools/releasetools
  
  # Setup device
  copy_directory device/generic
  copy_directory device/google/pesto
  
  # Link tools chains and related libs
  symlink_directory prebuilts/gcc
  symlink_directory bionic
  symlink_directory system/apex
  symlink_directory system/sepolicy
  symlink_directory system/tools/aidl
  symlink_directory system/tools/xsdc
  
  # Add TF, which is prebuilt due to there's too much dependencies need to adress, use prebuilt only first.
  symlink_directory tools/tradefederation/prebuilts
  
  # Add adb, source version is an apex module still need to figure out the lost part, use prebuilt verison also.
  symlink_directory prebuilts/runtime/adb
  
  # Add atest binaries.
  symlink_directory tools/asuite/
  symlink_directory prebuilts/asuite/atest
  
  # Add a host native test case
  symlink_directory platform_testing/tests/native
  symlink_directory external/libxml2
  symlink_directory external/libcxx
  symlink_directory external/googletest
  symlink_directory external/libcxxabi
  symlink_directory external/icu
  
}


function setup {
  cleanup_mock_top
  mkdir -p "$MOCK_TOP"

  echo
  echo ----------------------------------------------------------------------------
  info "Running test case \e[96;1m${FUNCNAME[1]}\e[0m"
  cd "$MOCK_TOP"

  tar xzf "$WARMED_UP_MOCK_TOP"
}

# shellcheck disable=SC2120
function run_soong {
  USE_RBE=false build/soong/soong_ui.bash --make-mode --skip-ninja --skip-config --soong-only --skip-soong-tests "$@"
}

function create_mock_bazel {
  copy_directory build/bazel
  copy_directory build/bazel_common_rules

  symlink_directory prebuilts/bazel
  symlink_directory prebuilts/clang
  symlink_directory prebuilts/jdk
  symlink_directory external/bazel-skylib
  symlink_directory external/bazelbuild-rules_android
  symlink_directory external/bazelbuild-rules_license
  symlink_directory external/bazelbuild-kotlin-rules

  symlink_file WORKSPACE
  symlink_file BUILD
}

function run_bazel {
  # Remove the ninja_build output marker file to communicate to buildbot that this is not a regular Ninja build, and its
  # output should not be parsed as such.
  rm -rf out/ninja_build

  build/bazel/bin/bazel "$@"
}

function run_ninja {
  build/soong/soong_ui.bash --make-mode --skip-config --soong-only --skip-soong-tests "$@"
}

info "Starting Soong integration test suite $(basename "$0")"
info "Mock top: $MOCK_TOP"


export ALLOW_MISSING_DEPENDENCIES=true
export ALLOW_BP_UNDER_SYMLINKS=true
warmup_mock_top
