#!/bin/bash -eu

set -o pipefail

# Test that relative symlinks work by recreating the bug in b/259191764

source "$(dirname "$0")/lib.sh"

function test_movable_top {
  setup

  mkdir -p a
  touch a/g.txt
  cat > a/Android.bp <<'EOF'
filegroup {
    name: "g",
    srcs: ["g.txt"],
    bazel_module: {bp2build_available: true},
  }
EOF

  # A directory under $MOCK_TOP
  outdir=out2

  # Modify OUT_DIR in a subshell so it doesn't affect the top level one.
  (export OUT_DIR=$outdir; run_soong bp2build && run_bazel build --config=bp2build --config=ci //a:g)

  # Move the output directory
  newoutdir=out3
  mv $outdir $newoutdir

  trap "rm -rf $newoutdir" EXIT
  
  # remove the bazel output base
  rm -rf $newoutdir/bazel/output_user_root
  (export OUT_DIR=$newoutdir; run_soong bp2build && run_bazel build --config=bp2build --config=ci //a:g)
}

scan_and_run_tests
