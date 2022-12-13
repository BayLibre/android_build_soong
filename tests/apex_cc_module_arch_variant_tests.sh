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

# Integration test for verifying arch variant cflags set on cc modules included
# in Bazel-built apexes in the real source tree.

if [ ! -e "build/make/core/Makefile" ]; then
  echo "$0 must be run from the top of the Android source tree."
  exit 1
fi

############
# Test Setup
############

OUTPUT_DIR="$(mktemp -d tmp.XXXXXX)"
BAZEL_OUTPUT_DIR="$OUTPUT_DIR/bazel"

export TARGET_PRODUCT="aosp_arm64"
[ "$#" -ge 1 ] && export TARGET_PRODUCT="$1"
ARCH_VARIANT_CFLAG="armv8-a"
[ "$#" -ge 2 ] && ARCH_VARIANT_CFLAG="$2"

function call_bazel() {
  build/bazel/bin/bazel --output_base="$BAZEL_OUTPUT_DIR" $@
}

function cleanup {
  # call bazel clean because some bazel outputs don't have w bits.
  call_bazel clean
  rm -rf "${OUTPUT_DIR}"
}
trap cleanup EXIT

######################
# Run bp2build / Bazel
######################
build/soong/soong_ui.bash --make-mode BP2BUILD_VERBOSE=1 --skip-soong-tests bp2build

# Number of CppCompile actions with arch variant
actions_num1=$(call_bazel aquery --config=bp2build --config=ci --config=android \
  'mnemonic("CppCompile", deps(//build/bazel/examples/apex/minimal:build.bazel.examples.apex.minimal.apex))' | grep -c "\-march=$ARCH_VARIANT_CFLAG")

# Number of all CppCompile actions
actions_num2=0
aquery_summary=$(call_bazel aquery --config=bp2build --config=ci --config=android --output=summary \
  'mnemonic("CppCompile", deps(//build/bazel/examples/apex/minimal:build.bazel.examples.apex.minimal.apex))' \
  | egrep -o '.*android_target.*: ([0-9]+)$' \
  | cut -d: -f2 -)

while read -r num;
do
  actions_num2=$(($actions_num2 + $num))
done <<< $aquery_summary

if [ $actions_num1 -eq $actions_num2 ]
then
  echo "Pass: arch variant is set."
else
  echo "Error: number of CppCompile actions with arch variant set: actual=$actions_num1, expected=$actions_num2"
  exit 1
fi