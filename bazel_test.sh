#!/bin/bash -eu

source build/soong/test_framework.sh

HARDWIRED_MOCK_TOP=
# Uncomment this to be able to view the source tree after a test is run
# HARDWIRED_MOCK_TOP=/tmp/td

function setup() {
  create_mock_top

  copy_directory build/bazel

  symlink_directory prebuilts/bazel
  symlink_directory prebuilts/jdk

  symlink_file WORKSPACE
  symlink_file tools/bazel
}


function test_smoke {
  setup

  tools/bazel info || fail "Bazel invocation failed"
}

run_test_suite
