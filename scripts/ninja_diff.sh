#!/bin/bash -eu
#
# Copyright 2017 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# This file makes it easy to confirm that a set of changes in source code don't result in any
# changes to the generated ninja files. This is to reduce the effort required to be confident
# in the correctness of refactorings

function die() {
  echo "$@" >&2
  exit 1
}

function usage() {
  violation="$1"
  die "Usage: ninja_diff.sh <oldVersions> <newVersions>

  For example:
  ninja_diff.sh 'build/soong:work^ build/blueprint:work^' 'build/soong:work build/blueprint:work'

  This file runs multiproduct_kati against both sets of versions and checks whether the ninja files
  changed between the two builds

  $violation"
}

#argument validation
if [ "$#" != "2" ]; then
  usage ""
fi
oldVersions="$1"
newVersions="$2"

# find some file paths
cd "$(dirname $0)"
SCRIPT_DIR="$PWD"
cd ../../..
CHECKOUT_ROOT="$PWD"
WORK_DIR="$PWD/diff"
OUT_DIR_OLD="$WORK_DIR/out_old"
OUT_DIR_NEW="$WORK_DIR/out_new"
OUT_DIR_TEMP="$WORK_DIR/out_temp"


function checkout() {
  versionSpecs="$1"
  for versionSpec in $versionSpecs; do
    project="$(echo $versionSpec | sed 's|\([^:]*\):\([^:]*\)|\1|')"
    ref="$(echo     $versionSpec | sed 's|\([^:]*\):\([^:]*\)|\2|')"
    echo "checking out ref $ref in project $project"
    cd "$CHECKOUT_ROOT/$project"
    git checkout "$ref"
  done
}

function run_build() {
  echo
  echo "Starting build"
  # rebuild multiproduct_kati, in case it was missing before,
  # or in case it is affected by some of the changes we're testing
  cd "$CHECKOUT_ROOT"
  make blueprint_tools
  # find multiproduct_kati and have it build the ninja files for each product
  builder="$(echo ./out/soong/host/*/bin/multiproduct_kati)"
  BUILD_NUMBER=sample "$builder" --keep=zip --out "$OUT_DIR_TEMP" || true
  echo
}

function diffProduct() {
  product="$1"

  dir1="$OUT_DIR_OLD/$product"
  zip1="$dir1/output.zip"
  unzipped1="$dir1/unzipped"

  dir2="$OUT_DIR_NEW/$product"
  zip2="$dir2/output.zip"
  unzipped2="$dir2/unzipped"
  unzip -qq "$zip1" -d "$unzipped1"
  unzip -qq "$zip2" -d "$unzipped2"

  #do a diff of the ninja files
  diffFile="$WORK_DIR/diff.txt"
  diff -r "$unzipped1" "$unzipped2" -x build_date.txt -x build_number.txt -x '\.*' -x '*.log' -x build_fingerprint.txt -x build.ninja.d '*.zip' | head -n 10 > "$diffFile"
  if [[ -s "$diffFile" ]]; then
    # outputs are different, so remove the unzipped versions but keep the zipped versions
    echo "Some differences for product $product:"
    cat "$diffFile"
    echo "End of differences for product $product"
    rm -rf "$unzipped1" "$unzipped2"
  else
    # outputs are the same, so remove all of the outputs
    rm -rf "$dir1" "$dir2"
  fi
}

function do_builds() {
  #reset work dir
  rm -rf "$WORK_DIR"
  mkdir "$WORK_DIR"

  #build new code
  checkout "$newVersions"
  run_build
  mv "$OUT_DIR_TEMP" "$OUT_DIR_OLD"

  #build old code
  #TODO do we want to cache old results? Maybe by the time we care to cache old results this will be running on a remote server somewhere and be completely different
  checkout "$oldVersions"
  run_build
  mv "$OUT_DIR_TEMP" "$OUT_DIR_NEW"

  #cleanup
  echo created "$OUT_DIR_OLD" and "$OUT_DIR_NEW"
}

function main() {
  do_builds
  checkout "$newVersions"

  #find all products
  productsFile="$WORK_DIR/all_products.txt"
  find $OUT_DIR_OLD $OUT_DIR_NEW -mindepth 2 -maxdepth 2 -name "output.zip" | sed "s|^$OUT_DIR_OLD/||" | sed "s|^$OUT_DIR_NEW/||" | sed "s|/output.zip$||" | sort | uniq > "$productsFile"
  echo Diffing products
  for product in $(cat $productsFile); do
    diffProduct "$product"
  done
  echo Done diffing products
  echo "Any outputs can be seen at $OUT_DIR_OLD and $OUT_DIR_NEW"
}

main
