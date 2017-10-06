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
# changes to the generated ninja files. This is to reduce the effort required to improve the
# confidence in the correctness of refactoring efforts

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
REPO_DIR="$CHECKOUT_ROOT/.repo"
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
  "$SCRIPT_DIR"/multiproduct_kati.bash --out "$OUT_DIR_TEMP"
  echo
}

function main() {
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
  checkout "$newVersions"

  #compute results
  diffFile="$WORK_DIR/diff.txt"
  diff -r "$OUT_DIR_OLD" "$OUT_DIR_NEW" -x soong.log -x build.trace.gz > "$diffFile" || true

  #show results
  echo "Changes in build output:"
  cat "$diffFile"
  echo "End of build differences."
  echo "Code changes that were tested:"
  echo "$oldVersions"
  echo "$newVersions"
  echo "End of code changes that were tested."
  echo "Outputs can be seen at $OUT_DIR_OLD and $OUT_DIR_NEW"
}

main
