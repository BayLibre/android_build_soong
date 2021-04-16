#!/bin/bash -eu

# This test exercises mixed builds where Soong and Bazel cooperate in building
# Android.
#
# When the execroot is deleted, the Bazel server process will automatically
# terminate itself.

source "$(dirname "$0")/lib.sh"

function create_mock_bazel() {
  copy_directory build/bazel

  symlink_directory prebuilts/bazel
  symlink_directory prebuilts/jdk

  symlink_file WORKSPACE
  symlink_file tools/bazel
}

function test_bazel_smoke {
  setup
  create_mock_bazel

  tools/bazel info
}

# This doesn't quite work because C++ implicit dependencies are not in the
# mock top.
function test_mixed_builds_integrated_bp2build_smoke {
  setup
  create_mock_bazel

  mkdir -p a
  touch a/a.txt
  cat > a/Android.bp <<'EOF'
filegroup {
  name: "a",
  srcs: ["a.txt"],
  bazel_module: { bp2build_available: true },
}
EOF

  INTEGRATED_BP2BUILD=1 USE_BAZEL_ANALYSIS=1 run_soong || fail "Build failed"
}

test_mixed_builds_integrated_bp2build_smoke
exit 0

test_bazel_smoke
test_mixed_builds_integrated_bp2build_smoke
