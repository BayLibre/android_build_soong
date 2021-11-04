#!/bin/bash -eu

set -o pipefail

TOP="$(readlink -f "$(dirname "$0")"/../../..)"
cd $TOP
build/soong/tests/bootstrap_test.sh
build/soong/tests/mixed_mode_test.sh
build/soong/tests/bp2build_bazel_test.sh
build/soong/tests/soong_test.sh
build/bazel/ci/rbc_product_config.sh aosp_arm64-userdebug
