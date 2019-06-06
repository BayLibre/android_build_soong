#!/bin/bash

# Copyright 2019 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Common code for developer helper scripts.

#####################################################################
# Print the message to stderr with the prefix ERROR and abort this
# script.
#####################################################################
function common::log_FATAL() {
  echo "ERROR:" "$*" >&2
  exit 1
}


#####################################################################
# Print the message to stderr with the prefix WARN
#####################################################################
function common::log_WARN() {
  echo "WARN:" "$*" >&2
}


#####################################################################
# Print the message with the prefix INFO.
#####################################################################
function common::log_INFO() {
  echo "INFO:" "$*"
}

#####################################################################
# Find the root project directory of this repo. This is done by
# finding the directory of where this script lives and then go up one
# directory to check the ".repo" directory exist. If not, keep going
# up until we find the ".repo" file or we reached to the filesystem
# root. Project root directory is printed to stdout.
#####################################################################
function common::root_dir() (
  local dir
  if ! dir="$("${readlink}" -e $(dirname "$0"))"; then
    log_FATAL "failed to read the script's current directory."
  fi

  dir=${dir}/../../..
  if ! dir="$("${readlink}" -e "${dir}")"; then
    log_FATAL "Cannot find the root project directory"
  fi

  echo "${dir}"
)

readonly os_type="$(uname -s)"
case "${os_type}" in
  Darwin)
    readlink=greadlink
    ;;
  Linux)
    readlink=readlink
    ;;
    *)
    common::log_FATAL "${os_type} is not a recognized system."
esac
readonly readlink


