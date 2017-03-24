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
cd $(dirname $0)
SCRIPT_PATH="$PWD"
cd ../../..

#parse args
lunchTarget="$1"
shift
makeInvocation="make $*"

#run
$SCRIPT_PATH/run_under_lunch.sh "$lunchTarget" "$makeInvocation"

