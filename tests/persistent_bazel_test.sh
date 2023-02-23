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

# This test verifies that adding USE_PERSISTENT_BAZEL creates a Bazel process
# that outlasts the build process.
# This test should only be run in sandboxed environments (because this test
# verifies a Bazel process using global process list, and may spawn lingering
# Bazel processes).
function test_persistent_bazel {
  setup

  # Ensure no existing Bazel process.
  if [[ -e out/bazel/output/server/server.pid.txt ]]; then
    cat out/bazel/output/server/server.pid.txt | xargs kill 2>/dev/null || true
    if cat out/bazel/output/server/server.pid.txt | grep ps -h ; then
      fail "Error killing pre-setup bazel"
    fi
  fi

  USE_PERSISTENT_BAZEL=1 run_soong nothing

  if cat out/bazel/output/server/server.pid.txt | grep ps -h ; then
    fail "Persistent bazel process expected, but not found after first build"
  fi

  USE_PERSISTENT_BAZEL=1 run_soong nothing

  if cat out/bazel/output/server/server.pid.txt | grep ps -h ; then
    fail "Persistent bazel process expected, but not found after second build"
    cat out/bazel/output/server/server.pid.txt | xargs kill 2>/dev/null
    if cat out/bazel/output/server/server.pid.txt | grep ps -h ; then
      fail "Error killing bazel on shutdown"
    fi
  fi
}

scan_and_run_tests
