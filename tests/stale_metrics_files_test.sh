#!/bin/bash

# This test ensures that stale metrics files are deleted after each run

# Run bazel
# Note - bp2build metrics are present after clean runs, only
USE_RBE=false build/soong/soong_ui.bash --make-mode clean
build/bazel/bin/b build libcore:all

# ensure bazel metrics file is present
if [[ ! -f out/bazel_metrics.pb || ! -f out/build_progress.pb || ! -f out/soong_metrics || ! -f out/bp2build_metrics.pb  ]]; then
   echo "Missing metrics files for b run"
   exit 1
fi

# Run a soong build
build/soong/soong_ui.bash --make-mode nothing

# Ensure all proper metrics files are present
if [[ ! -f out/soong_build_metrics.pb || ! -f out/build_progress.pb || ! -f out/soong_metrics || ! -f out/bp2build_metrics.pb || ! -f out/rbe_metrics.pb ]]; then
   ls out/soong_build_metrics out/build_progress.pb out/soong_metrics out/bp2build_metrics.pb out/rbe_metrics.pb
   echo "Missing metrics files for soong run"
   exit 1
fi


# Ensure that bazel_metrics.pb is deleted
if [[ -f out/bazel_metrics.pb ]]; then
   echo "Stale bazel metrics file detected"
   exit 1
fi

# Run bazel again - to make sure that soong_build_metrics.pb gets deleted
build/bazel/bin/b build libcore:all

if [[ -f out/soong_build_metrics.pb ]]; then
   echo "Stale soong build metrics file detected"
   exit 1
fi
