#!/bin/bash

# This test ensures that stale metrics files are deleted after each run

# Run bazel
build/bazel/bin/b build libcore:all

# Run a soong build
USE_RBE=false build/soong/soong_ui.bash --make-mode nothing

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
