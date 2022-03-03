#!/bin/bash -eu

set -o pipefail

if [[ -z ${OUT_DIR+x} ]]; then
  echo "OUT_DIR not set. Using out. This should only be used for manual developer testing."
  OUT_DIR=out
fi

#if [[ -z ${DIST_DIR+x} ]]; then
#  DIST_DIR="$OUT_DIR/dist"
#  print "DIST_DIR not set. Using $DIST_DIR. This should only be used for manual developer testing.\n"
#fi

TARGETS=(
  libc
)

TARGET_PRODUCT=aosp_arm64
TARGET_BUILD_VARIANT=userdebug
NINJA_PATH="$OUT_DIR/combined-$TARGET_PRODUCT.ninja"
LEGACY_OBJ="$OUT_DIR/target/product/generic_arm64/apex/com.android.runtime/lib/bionic/libc.so"
BAZEL_OBJ=$LEGACY_OBJ

build/soong/soong_ui.bash --make-mode \
  TARGET_PRODUCT=$TARGET_PRODUCT \
  TARGET_BUILD_VARIANT=$TARGET_BUILD_VARIANT\
  "${TARGETS[@]}"
mkdir -p "$OUT_DIR/diffing/legacyBuildFiles"
build/bazel/scripts/difftool/collect.py "$NINJA_PATH" "$OUT_DIR/diffing/legacyBuildFiles" --file="$LEGACY_OBJ"

build/soong/soong_ui.bash --make-mode \
  BP2BUILD_VERBOSE=1 \
  USE_BAZEL_ANALYSIS=1 \
  BAZEL_STARTUP_ARGS="--max_idle_secs=5" \
  BAZEL_BUILD_ARGS="--color=no --curses=no --show_progress_rate_limit=5" \
  TARGET_PRODUCT=$TARGET_PRODUCT \
  TARGET_BUILD_VARIANT=$TARGET_BUILD_VARIANT
mkdir -p "$OUT_DIR/diffing/bazelBuildFiles"
build/bazel/scripts/difftool/collect.py "$NINJA_PATH" "$OUT_DIR/diffing/bazelBuildFiles" --file="$BAZEL_OBJ"

build/bazel/scripts/difftool/difftool.py "$OUT_DIR/diffing/legacyBuildFiles" "$OUT_DIR/diffing/bazelBuildFiles"
