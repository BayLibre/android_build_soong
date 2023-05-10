#!/bin/bash -eu

# Copyright (C) 2023 The Android Open Source Project
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

set -o pipefail

source "$(dirname "$0")/lib.sh"

# This test verifies that adding a new glob to a blueprint file only
# causes build.ninja to be regenerated on the *next* build, and *not*
# the build after. (This is a regression test for a bug where globs
# resulted in two successive regenerations.)
function test_new_glob_incrementality {
  setup

  run_soong nothing
  local -r mtime1=$(stat -c "%y" out/soong/build.ninja)

  mkdir -p globdefpkg/
  cat > globdefpkg/Android.bp <<'EOF'
filegroup {
  name: "fg_with_glob",
  srcs: ["*.txt"],
}
EOF

  run_soong nothing
  local -r mtime2=$(stat -c "%y" out/soong/build.ninja)

  if [[ "$mtime1" == "$mtime2" ]]; then
    fail "Ninja file was not regenerated, despite a new bp file"
  fi

  run_soong nothing
  local -r mtime3=$(stat -c "%y" out/soong/build.ninja)

  if [[ "$mtime2" != "$mtime3" ]]; then
    fail "Ninja file was regenerated despite no previous bp changes"
  fi
}

scan_and_run_tests
