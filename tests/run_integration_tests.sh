#!/bin/bash -eu

set -o pipefail

TARGETS=(
  py3-cmd
  libc
)

TARGET_PRODUCT=aosp_arm64
TARGET_BUILD_VARIANT=userdebug
NINJA_PATH=combined-$TARGET_PRODUCT.ninja
LEGACY_OBJ=target/product/generic_arm64/apex/com.android.runtime/lib/bionic/libc.so
BAZEL_OBJ=$LEGACY_OBJ

PY=out/host/linux-x86/bin/py3-cmd
# soong build
export OUT_DIR=out/1
build/soong/soong_ui.bash --make-mode \
  TARGET_PRODUCT=$TARGET_PRODUCT \
  TARGET_BUILD_VARIANT=$TARGET_BUILD_VARIANT \
  "${TARGETS[@]}"
mkdir -p "$OUT_DIR/diff"
$PY build/bazel/scripts/difftool/collect.py "$OUT_DIR/$NINJA_PATH" "$OUT_DIR/diff" --file="$OUT_DIR/$LEGACY_OBJ"


# bazel build
export OUT_DIR=out/2
build/soong/soong_ui.bash --make-mode \
  BP2BUILD_VERBOSE=1 \
  USE_BAZEL_ANALYSIS=1 \
  BAZEL_STARTUP_ARGS="--max_idle_secs=5" \
  BAZEL_BUILD_ARGS="--color=no --curses=no --show_progress_rate_limit=5" \
  TARGET_PRODUCT=$TARGET_PRODUCT \
  TARGET_BUILD_VARIANT=$TARGET_BUILD_VARIANT \
  "${TARGETS[@]}"
mkdir -p "$OUT_DIR/diff"
$PY build/bazel/scripts/difftool/collect.py "$OUT_DIR/$NINJA_PATH" "$OUT_DIR/diff" --file="$OUT_DIR/$BAZEL_OBJ"

# diff soong and bazel outputs
$PY build/bazel/scripts/difftool/difftool.py "out/1/diff" "out/2/diff"
