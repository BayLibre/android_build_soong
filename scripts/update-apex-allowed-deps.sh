#!/bin/bash -e
#
# The script to run locally to re-generate global allowed list of dependencies
# for updatable modules.

if [ ! -e "build/envsetup.sh" ]; then
  echo "ERROR: $0 must be run from the top of the tree"
  exit 1
fi

source build/envsetup.sh > /dev/null || exit 1

readonly ALLOWED_DEPS_FILE="build/soong/apex/allowed_deps.txt"
readonly OUT_DIR=$(get_build_var OUT_DIR)
readonly FILTERED_UPDATABLE_FLATLISTS="${OUT_DIR}/soong/apex/depsinfo/filtered-updatable-flatlists.txt"

# If the script is run after droidcore failure, ${FILTERED_UPDATABLE_FLATLISTS}
# should already be built. If running the script manually, make sure it exists.
m "${FILTERED_UPDATABLE_FLATLISTS}"

# Preserve existing comment from the allowed_deps.txt and append new flatlists
comment_file="$(mktemp)"
grep '^#' "${ALLOWED_DEPS_FILE}" > "${comment_file}"
cat "${comment_file}" "${FILTERED_UPDATABLE_FLATLISTS}" > "${ALLOWED_DEPS_FILE}"

# Clean up temp files
rm "${comment_file}"
