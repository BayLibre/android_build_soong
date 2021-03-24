#!/bin/bash -eu

function fail {
  echo ERROR: $1
  exit 1
}

function setup() {
  MOCK_TOP=$(mktemp -t -d st.XXXXX)
  trap 'echo cd / && echo rm -fr "$MOCK_TOP"' EXIT

  echo "Test case: ${FUNCNAME[1]}, mock top path: $MOCK_TOP"
  cd "$MOCK_TOP"

  copy_directory build/blueprint
  copy_directory build/soong

  symlink_directory prebuilts/go
  symlink_directory prebuilts/build-tools
  symlink_directory external/golang-protobuf

  touch "$MOCK_TOP/Android.bp"

  export ALLOW_MISSING_DEPENDENCIES=true

  mkdir -p out/soong
}

function run_soong() {
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
}
