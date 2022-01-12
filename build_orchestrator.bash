#!/bin/bash

set -euo pipefail

# Help
# In your inner tree, run (update params accordingly)
# INNER_TREES=../master,../master-art MERGED_CDK_DIR=../out/cdk build/soong/build_orchestrator.bash
# <target>

# Steps
# 1. In each inner tree, build the "cdk". This creates files in
# out/soong/.export  of the inner tree
# 2. Coalesce the out/soong/.export of each inner tree
# 3. Mount the coalesced dir to out/soong/.import in RO mode for m
# <target>

# Convert space separated cli args to comma separated
# TODO: This should probably be dynamicaly determined from some manifest file
INNER_TREES=$(echo $INNER_TREES | tr ',' ' ')

function build_cdk() {
  (cd $1 && ALLOW_MISSING_DEPENDENCIES=true run_soong cdk)
}

function run_soong() {
  build/soong/soong_ui.bash --make-mode $@
}

# Step 1: Build the cdk
# Sequential now, but can be run in parallel
for inner_tree in $INNER_TREES; do
  build_cdk $inner_tree
done

# Step 2: Merge the cdk files from each inner tree
mkdir -p $MERGED_CDK_DIR
for inner_tree in $INNER_TREES; do
  rsync -a $inner_tree/out/soong/.export/ $MERGED_CDK_DIR
done

# Step 3: Build the component
# In your inner tree, run MERGED_CDK_DIR=<> m <component>
# TODO: Remove ALLOW_MISSING_DEPENDENCIES, it is not needed in the "primary" build
CDK_DIR=$(realpath $MERGED_CDK_DIR) ALLOW_MISSING_DEPENDENCIES=true exec build/soong/soong_ui.bash --make-mode $@
