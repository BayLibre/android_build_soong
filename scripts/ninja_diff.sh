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
  die "$violation

  Usage: ninja_diff.sh [--products=product1,product2...] <OLD_VERSIONS> <NEW_VERSIONS>

  This file builds and parses the build (Android.mk, Android.bp, etc) files for each requested
  product and for both sets of versions, and checks whether the ninja files (which implement
  the build graph) changed between the two versions.

  Example: ninja_diff.sh 'build/soong:work^ build/blueprint:work^' 'build/soong:work build/blueprint:work'

  Options:
    --products=PRODUCTS  comma-separated list of products to check

  $violation"
}

PRODUCTS_ARG=""
OLD_VERSIONS=""
NEW_VERSIONS=""
function parse_args() {
  # parse optional arguments
  while true; do
    arg="$1"
    case "$arg" in
      --products=*) PRODUCTS_ARG="$arg";;
      *) return;;
    esac
    shift
  done
  # parse required arguments
  if [ "$#" != "2" ]; then
    usage ""
  fi
  #argument validation
  OLD_VERSIONS="$1"
  NEW_VERSIONS="$2"

}
parse_args "$@"


# find some file paths
cd "$(dirname $0)"
SCRIPT_DIR="$PWD"
cd ../../..
CHECKOUT_ROOT="$PWD"
OUT_DIR="${OUT_DIR-}"
if [ -z "$OUT_DIR" ]; then
  OUT_DIR=out
fi
WORK_DIR="$OUT_DIR/diff"
OUT_DIR_OLD="$WORK_DIR/out_old"
OUT_DIR_NEW="$WORK_DIR/out_new"
OUT_DIR_TEMP="$WORK_DIR/out_temp"


function checkout() {
  versionSpecs="$1"
  for versionSpec in $versionSpecs; do
    project="$(echo $versionSpec | sed 's|\([^:]*\):\([^:]*\)|\1|')"
    ref="$(echo     $versionSpec | sed 's|\([^:]*\):\([^:]*\)|\2|')"
    echo "checking out ref $ref in project $project"
    git -C "$CHECKOUT_ROOT/$project" checkout "$ref"
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
  builder="$(echo $OUT_DIR/soong/host/*/bin/multiproduct_kati)"
  BUILD_NUMBER=sample "$builder" $PRODUCTS_ARG --keep=zip --out "$OUT_DIR_TEMP" || true
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
  diff -r "$unzipped1" "$unzipped2" -x build_date.txt -x build_number.txt -x '\.*' -x '*.log' -x build_fingerprint.txt -x build.ninja.d -x '*.zip' | head -n 10 > "$diffFile"
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
  checkout "$NEW_VERSIONS"
  run_build
  mv "$OUT_DIR_TEMP" "$OUT_DIR_NEW"

  #build old code
  #TODO do we want to cache old results? Maybe by the time we care to cache old results this will
  #be running on a remote server somewhere and be completely different
  checkout "$OLD_VERSIONS"
  run_build
  mv "$OUT_DIR_TEMP" "$OUT_DIR_OLD"

  #cleanup
  echo created "$OUT_DIR_OLD" and "$OUT_DIR_NEW"
}

function main() {
  do_builds
  checkout "$NEW_VERSIONS"

  #find all products
  productsFile="$WORK_DIR/all_products.txt"
  find $OUT_DIR_OLD $OUT_DIR_NEW -mindepth 2 -maxdepth 2 -name "output.zip" | sed "s|^$OUT_DIR_OLD/||" | sed "s|^$OUT_DIR_NEW/||" | sed "s|/output.zip$||" | sort | uniq > "$productsFile"
  echo Diffing products
  for product in $(cat $productsFile); do
    diffProduct "$product"
  done
  echo Done diffing products
  echo "Any differing outputs can be seen at $OUT_DIR_OLD/*/output.zip and $OUT_DIR_NEW/*/output.zip"
}

main
