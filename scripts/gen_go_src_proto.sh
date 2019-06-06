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

# Generate the go src from a proto file. The go src file is placed in the same
# locate as the proto file. At this point, only linux based is supported.

set -e

source "$(cd "$(dirname $0)" ; pwd -P)/common.sh"

#####################################################################
# Build the aprotoc (and it's dependencies) using the soong build
# system. aprotoc is the application to go the conversion from proto
# file to go src.
#####################################################################
function build_aprotoc() (
  local -r soong_ui="build/soong/soong_ui.bash"
  local -r modules=("aprotoc" "protoc-gen-go")

  cd "${TOP_DIR}"
  if ! "${soong_ui}" --make-mode ${modules[@]}; then
    common::log_FATAL "Failed to build aprotoc. See out/error.log for further details."
  fi
)


#####################################################################
# Returns the path of built's aprotoc.
#####################################################################
function aprotoc() (
  echo "${TOP_DIR}/out/soong/host/${PLATFORM_DIR}/bin/aprotoc"
)


#####################################################################
# Returns the path of protoc-gen-go, a plugin for aprotoc for
# generating go src from proto file.
#####################################################################
function protoc-gen-go() (
  echo "${TOP_DIR}/out/soong/host/${PLATFORM_DIR}/bin/protoc-gen-go"
)


#####################################################################
# Generate the go src file from the proto file. Assumes that aprotoc
# has been built.
#####################################################################
function generate_src_file() (
  if ! proto_file_dir=$(dirname "${PROTO_FILE_ARG}"); then
    common::log_FATAL "Unable to retrieve the full path of ${PROTO_FILE_ARG}"
  fi
  readonly proto_file_dir
  cd "${proto_file_dir}"

  if ! proto_filename=$(basename "${PROTO_FILE_ARG}"); then
    common::log_FATAL "Failed to get the basename of ${PROTO_FILE_ARG}"
  fi
  readonly proto_filename

  if $(aprotoc) --plugin=$(protoc-gen-go) --go_out="paths=source_relative:." ${proto_filename}; then
    common::log_INFO "Successfuly generated the go src of ${proto_filename}."
  else
    common::log_FATAL "Failed to generate the go src of ${proto_filename}."
  fi
)


#####################################################################
# Help function.
#####################################################################
function help() (
  cat <<EOF
Generates the go source file from a proto file. Requires the
following arguments to be defined:

  --proto-file=<filename>
  The path of the proto file.

The resulting go src generation is placed in the same directory
where the proto-file resides.

Example:

./$(basename $0) --proto-file=build/soong/ui/metrics/metrics_proto/metrics.proto

EOF
)


#####################################################################
# Parse the arguments passed in to this script.
#####################################################################
function parse_arguments() {
  local proto_file=""
  while [[ -n "$1" ]]; do
    case "$1" in
      --proto-file=*)
        proto_file="${1#*=}"
        shift
        ;;
      --help)
        help
        shift
        exit 0
        ;;
      *)
        common::log_WARN "Unknown option: $1"
        help
        exit 1
        ;;
    esac
  done

  if [[ -z "${proto_file}" ]]; then
    common::log_FATAL "Please specify --proto-file."
  fi
  if [[ ! -e "${proto_file}" ]]; then
    common::log_FATAL "\"${proto_file}\" does not exist."
  fi
  if ! PROTO_FILE_ARG="$(${readlink} -f "${proto_file}")"; then
    common::log_FATAL "Failed to get the full path of ${proto_file}"
  fi
  readonly PROTO_FILE_ARG
}


function main() {
  parse_arguments "$@"
  build_aprotoc
  generate_src_file
}


if ! TOP_DIR="$(common::root_dir)"; then
  common::log_FATAL "Failed to find the root of the repo checkout"
fi
readonly TOP_DIR

case "${os_type}" in
  Linux)
  PLATFORM_DIR="linux-x86"
  readonly PLATFORM_DIR
  ;;
  *)
  common::log_FATAL "${os_type} is not a recognized system."
  ;;
esac


main "$@"
