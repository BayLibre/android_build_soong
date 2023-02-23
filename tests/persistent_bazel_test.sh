#!/bin/bash -eu

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
  pidof "bazel(workspace)" | xargs kill 2>/dev/null

  if ps aux | grep -v grep | grep "bazel(workspace)" ; then
    fail "Persistent bazel process cannot run with already-live Bazel"
  fi

  USE_PERSISTENT_BAZEL=1 run_soong nothing

  if ! ps aux | grep -v grep | grep "bazel(workspace)" ; then
    fail "Persistent bazel process expected, but not found after first build"
  fi

  USE_PERSISTENT_BAZEL=1 run_soong nothing

  if ! ps aux | grep -v grep | grep "bazel(workspace)" ; then
    fail "Persistent bazel process expected, but not found after second build"
  fi

  pidof "bazel(workspace)" | xargs kill 2>/dev/null

  if ps aux | grep -v grep | grep "bazel(workspace)" ; then
    fail "Persistent bazel process not shut down after shutdown commandl"
  fi
}

scan_and_run_tests
