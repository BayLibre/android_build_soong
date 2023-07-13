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

set -uo pipefail

source "$(dirname "$0")/common.sh"

if [ ! -e "build/make/core/Makefile" ]; then
  echo "$0 must be run from the top of the Android source tree."
  exit 1
fi

# Integration tests for verifying some compliance related functions.

function test_use_product_out_in_meta_Lic_files {
  # Setup
  local out_dir
  out_dir=$(setup)
  trap "cleanup ${out_dir}" EXIT

  # Test
  # m droid
  run_soong "aosp_cf_x86_64_phone" "${out_dir}" "droid"

  # Check if there is any meta_lic or .meta_lic file still using value of PRODUCT_OUT in paths instead of {PRODUCT_OUT}
  local num_meta_lic
  num_meta_lic=$(find $out_dir \( -iname meta_lic -o -iname '*.meta_lic' \) -type f -exec grep -l \"$out_dir/target/product/vsoc_x86_64  '{}'  \; | wc -l)

  if [ $num_meta_lic != "0" ]
  then
    echo "Found $num_meta_lic meta_lic files that have not been converted to using {PRODUCT_OUT}."
    exit 1
  fi

  # Teardown
  cleanup "${out_dir}"
  echo PASS
}


scan_and_run_tests