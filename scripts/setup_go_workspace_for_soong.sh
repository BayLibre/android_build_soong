#!/bin/bash
set -e

#mounts the components of soong into a directory structure that Go tools and editors expect

#move to the script's directory
cd "$(dirname $0)"
SCRIPT_PATH="$PWD"

#find the root of the Repo checkout
cd "${SCRIPT_PATH}"/../../..
ANDROID_PATH="${PWD}"
cd ..
PARENT_PATH="${PWD}"
if [ -z "$GOPATH" ]; then
  echo 'Error, GOPATH must be nonempty. The generated workspace will be placed at ${GOPATH}'
  exit 1
else
  OUTPUT_PATH="${GOPATH}"
fi

function confirm() {
  while true; do
    echo "Will create GOPATH-compatible directory structure at ${OUTPUT_PATH}"
    echo -n "Ok [y/n]?"
    read decision
    if [ "${decision}" == "y" ]; then
      return 0
    else
      if [ "${decision}" == "n" ]; then
        return 1
      else
        echo "Invalid choice ${decision}; choose either 'y' or 'n'"
      fi
    fi
  done
}

function bindAll() {
  bindOne "${ANDROID_PATH}/build/blueprint" "${OUTPUT_PATH}/src/github.com/google/blueprint"
  bindOne "${ANDROID_PATH}/build/soong" "${OUTPUT_PATH}/src/android/soong"

  bindOne "${ANDROID_PATH}/art/build" "${OUTPUT_PATH}/src/android/soong/art"
  bindOne "${ANDROID_PATH}/external/llvm/soong" "${OUTPUT_PATH}/src/android/soong/llvm"
  bindOne "${ANDROID_PATH}/external/clang/soong" "${OUTPUT_PATH}/src/android/soong/clang"
  echo
  echo "Created GOPATH-compatible directory structure at ${OUTPUT_PATH}"
}

function bindOne() {
  #causes $newPath to mirror $existingPath
  existingPath="$1"
  newPath="$2"
  mkdir -p "$newPath"
  echoAndDo bindfs "${existingPath}" "${newPath}"
}

function echoAndDo() {
  echo "$@"
  eval "$@"
}

if confirm; then
  bindAll
else
  echo "skipping due to user request"
  exit 1
fi
