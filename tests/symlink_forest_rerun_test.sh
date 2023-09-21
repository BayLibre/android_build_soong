#!/bin/bash -eu

set -o pipefail

# Tests that symlink_Forest will rerun if soong_build has schanged

source "$(dirname "$0")/lib.sh"

function test_symlink_forest_reruns {
  setup

  mkdir -p a
  touch a/g.txt
  touch a/h.txt
  cat > a/Android.bp <<'EOF'
filegroup {
    name: "g",
    srcs: ["g.txt"],
  }
EOF

  run_soong g

  hash=`cat out/soong/workspace/soong_build_hash`
  # rerun with no changes - ensure that it hasn't changed
  run_soong g
  newhash=`cat out/soong/workspace/soong_build_hash`
  if [[ ! "$hash" == "$newhash" ]]; then
     fail "symlink forest reran when it shouldn't have"
  fi

  # change exit codes to force a soong_build rebuild.
  sed -i 's/os.Exit(1)/os.Exit(2)/g' build/soong/bp2build/symlink_forest.go

  run_soong g
  newhash=`cat out/soong/workspace/soong_build_hash`
  if [[ "$hash" == "$newhash" ]]; then
     fail "symlink forest did not rerun when it should have"
  fi

}

scan_and_run_tests
