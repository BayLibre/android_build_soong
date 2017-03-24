#!/bin/bash

set -e

function usage() {
  programName="$(basename $0)"
  echo "$programName"
  echo
  echo "Runs a certain command using the environment variables from a certain \`lunch\` invocation"
  echo "Doesn't require that envsetup or lunch were run previously."
  echo
  echo "usage  : \`$programName <lunch-target> '<command to run>'\`"
  echo "example: \`$programName aosp-arm_eng 'echo $ANDROID_BUILD_TOP'\`"
  exit 1
}

if [ -z "$1" ]; then
  usage
fi

#cd to repo root
WORKING_DIR="$PWD"
cd $(dirname $0)/../../..
REPO_ROOT="$PWD"

function echoAndDo() {
  echo "$@"
  eval "$@"
}

function sourceEnvSetup() {
  cd "${REPO_ROOT}/build"
  echoAndDo source envsetup.sh
}

function runLunch() {
  target="$1"
  echoAndDo lunch "$target"
}

function runCommand() {
  cd "$WORKING_DIR"
  echoAndDo $1 #this splits the input argument into multiple arguments
}


sourceEnvSetup
runLunch "$1"
runCommand "$2"
