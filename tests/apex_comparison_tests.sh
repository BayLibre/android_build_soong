#!/bin/bash

# Copyright (C) 2022 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

# Soong/Bazel integration test for building unbundled apexes in the real source tree.
#
# These tests build artifacts from head and compares their contents.

if [ ! -e "build/make/core/Makefile" ]; then
  echo "$0 must be run from the top of the Android source tree."
  exit 1
fi

output_dir="$(mktemp -d)"
soong_output_dir="$output_dir/soong"
bazel_output_dir="$output_dir/bazel"

function cleanup {
  rm -rf "${output_dir}"
}

trap cleanup EXIT

# Run Soong
export UNBUNDLED_BUILD_SDKS_FROM_SOURCE=true # don't rely on prebuilts
export TARGET_BUILD_APPS="com.android.adbd com.android.tzdata build.bazel.examples.apex.minimal"
packages/modules/common/build/build_unbundled_mainline_module.sh --product module_arm --dist_dir "$soong_output_dir"

# Run Bazel
build/soong/soong_ui.bash --make-mode BP2BUILD_VERBOSE=1 --skip-soong-tests bp2build
tools/bazel --output_base="$bazel_output_dir"  \
  build --config=bp2build --config=ci --config=android_arm \
  //packages/modules/adb/apex:com.android.adbd \
  //system/timezone/apex:com.android.tzdata \
  //build/bazel/examples/apex/minimal:build.bazel.examples.apex.minimal.apex

function run_deapexer() {
  tools/bazel run --config=bp2build --config=linux_x86_64 //system/apex/tools:deapexer -- "$@"
}

function compare_deapexer_list() {
  # Compare the outputs of `deapexer list`, which lists the contents of the apex filesystem image.
  local soong_apex=$1
  local bazel_apex=$2

  local soong_list="$output_dir/soong.list"
  local bazel_list="$output_dir/bazel.list"

  run_deapexer list "$soong_apex" > "$soong_list"
  run_deapexer list "$bazel_apex" > "$bazel_list"

  if cmp -s $soong_list $bazel_list
  then
    echo "ok"
  else
    echo expected
    echo
    cat "$soong_list"
    echo
    echo got
    echo
    cat "$bazel_list"
    exit 1
  fi
}

compare_deapexer_list \
  "$soong_output_dir/com.android.adbd.apex" \
  "$bazel_output_dir/execroot/__main__/bazel-out/android_arm-fastbuild/bin/packages/modules/adb/apex/com.android.adbd.apex"

compare_deapexer_list \
  "$soong_output_dir/com.android.tzdata.apex" \
  "$bazel_output_dir/execroot/__main__/bazel-out/android_arm-fastbuild/bin/system/timezone/apex/com.android.tzdata.apex"

compare_deapexer_list \
  "$soong_output_dir/build.bazel.examples.apex.minimal.apex" \
  "$bazel_output_dir/execroot/__main__/bazel-out/android_arm-fastbuild/bin/build/bazel/examples/apex/minimal/build.bazel.examples.apex.minimal.apex"
