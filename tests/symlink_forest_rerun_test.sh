#!/bin/bash -eu

set -o pipefail

# Test that symlink forest will overwrite stale symlinks

source "$(dirname "$0")/lib.sh"

function test_symlink_regeneration {
  setup

  mkdir -p a
  touch a/g.txt
  cat > a/Android.bp <<'EOF'
filegroup {
    name: "g",
    srcs: ["g.txt"],
  }
EOF

  # A directory under /tmp/...
  outdir='out2'
  # Modify OUT_DIR in a subshell so it doesn't affect the top level one.
  (export OUT_DIR=$outdir; run_soong g)
  
  # remove the symlink to bazel and replace it with a bogus symlink
  rm $outdir/soong/workspace/build/bazel/bin
  ln -s /tmp $outdir/soong/workspace/build/bazel/bin
  
  #remove a marker file to force reexecution
  rm $outdir/soong/bp2build_files_marker
  (export OUT_DIR=$outdir; run_soong g)
  
  bazel_path=`readlink $outdir/soong/workspace/build/bazel/bin`
  if [[ "$bazel_path" != "$MOCK_TOP/build/bazel/bin" ]]; then
     fail "Symlink was not regenerated"
  fi
}

scan_and_run_tests
