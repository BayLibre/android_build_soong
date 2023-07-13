#!/bin/bash

# Copyright (C) 2023 The Android Open Source Project
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

function setup {
  local tmp_dir
  tmp_dir="$(mktemp -d tmp.XXXXXX)"
  echo "${tmp_dir}"
}

function cleanup {
  local tmp_dir="$1"; shift
  if [ -e "${tmp_dir}" ]
  then
    rm -rf "${tmp_dir}"
    echo Removed "${tmp_dir}"
  fi
}

function run_soong {
  local -r target_product="$1";shift
  local -r out_dir="$1"; shift
  local targets="$1"; shift
  if [ "$#" -ge 1 ]; then
    local -r apps=$1; shift
    TARGET_PRODUCT="${target_product}" TARGET_BUILD_VARIANT=userdebug OUT_DIR="${out_dir}" TARGET_BUILD_UNBUNDLED=true TARGET_BUILD_APPS=$apps build/soong/soong_ui.bash --make-mode ${targets}
  else
    TARGET_PRODUCT="${target_product}" TARGET_BUILD_VARIANT=userdebug OUT_DIR="${out_dir}" build/soong/soong_ui.bash --make-mode ${targets}
  fi
}

function info {
  echo -e "\e[92;1m[TEST HARNESS INFO]\e[0m" "$*"
}

function fail {
  echo -e "\e[91;1mFAILED:\e[0m" "$*"
  exit 1
}

function scan_and_run_tests {
  # find all test_ functions
  # NB "declare -F" output is sorted, hence test order is deterministic
  readarray -t test_fns < <(declare -F | sed -n -e 's/^declare -f \(test_.*\)$/\1/p')
  info "Found ${#test_fns[*]} tests"
  if [[ ${#test_fns[*]} -eq 0 ]]; then
    fail "No tests found"
  fi
  for f in ${test_fns[*]}; do
    info "Start test case \e[96;1m$f\e[0m"
    $f
    info "Completed test case \e[96;1m$f\e[0m"
  done
}
