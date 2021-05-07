#! /bin/bash
# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -eu
declare -r build_dir=/tmp/rblftest
declare -r source_tree=$(realpath ../../..)
declare compile=t
while getopts "n" opt; do
  case $opt in
    n) compile= ;;
  esac
done
shift $(($OPTIND-1))
declare -r top_mk=$1
declare -r top_pcm=$build_dir/${top_mk/.mk/.rbc}

[[ -z "$compile" ]] || go run cmd/mk2rbc.go  -d $source_tree  -r --mode=write --outdir=$build_dir $top_mk
env UNAME="$(uname -sm)" TARGET_PRODUCT=aosp_arm TARGET_BUILD_VARIANT=userdebug TARGET_ARCH=armv8 rbcrun -d $source_tree  RBC_OUT="make,global" <(cat <<EOF
load("//build/make/core:product_config.rbc", "rblf")
load("$top_pcm", "init")
globals, config = rblf.product_configuration("test/device", init)
rblf.printvars(globals, config)
EOF
)