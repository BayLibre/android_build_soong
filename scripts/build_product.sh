#!/bin/bash

set -e

function usage() {
  programName="$(basename $0)"
  echo 'This script does a build of Android'
  echo
  echo 'This script reads the lunch-target from the command-line, removing the need to remember to rerun `lunch`'
  echo '  when changing directories to a separate checkout or upon needing a different lunch target.'
  echo
  echo "usage  : \`$programName <lunch-target> <make-args>\`"
  echo "example: \`$programName aosp_arm-eng -j\`"
  exit 1
}

if [ -z "$1" ]; then
  usage
fi

#cd to repo root
cd $(dirname $0)/../..
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

function runBuild() {
  cd "$REPO_ROOT"
  echoAndDo make "$@"
}


sourceEnvSetup
runLunch "$1"
shift
runBuild "$@"
