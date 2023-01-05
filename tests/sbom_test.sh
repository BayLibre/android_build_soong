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

# Integration test for verifying generated SBOM for cuttlefish device.

if [ ! -e "build/make/core/Makefile" ]; then
  echo "$0 must be run from the top of the Android source tree."
  exit 1
fi

tmp_dir="$(mktemp -d tmp.XXXXXX)"
function cleanup {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

# m droid
#out_dir=$tmp_dir
out_dir=out
TARGET_PRODUCT="aosp_cf_x86_64_phone" TARGET_BUILD_VARIANT=userdebug OUT_DIR=$out_dir build/soong/soong_ui.bash --make-mode dump.erofs sbom

# Generate installed file list from .img files in PRODUCT_OUT
dump_erofs=$out_dir/host/linux-x86/bin/dump.erofs
product_out=$out_dir/target/product/vsoc_x86_64
for f in $(ls $product_out/product.img $product_out/system_*.img); do
  partition_name=$(basename $f | cut -d. -f1)
  file_list_file="${product_out}/sbom-${partition_name}-files.txt"
  files_in_spdx_file="${product_out}/sbom-${partition_name}-files-in-spdx.txt"
  rm "$file_list_file" > /dev/null 2>&1
  all_dirs="/"
  while [ ! -z "$all_dirs" ]; do
    dir=$(echo "$all_dirs" | cut -d ' ' -f1)
    all_dirs=$(echo "$all_dirs" | cut -d ' ' -f1 --complement -s)
    entries=$($dump_erofs --ls --path "$dir" $f | tail -n +11)
    while read -r entry; do
      nid=$(echo $entry | sed 's/^\s*//' | cut -d ' ' -f1)
      entry=$(echo $entry | sed 's/^\s*//' | cut -d ' ' -f1 --complement)
      type=$(echo $entry | sed 's/^\s*//' | cut -d ' ' -f1)
      entry=$(echo $entry | sed 's/^\s*//' | cut -d ' ' -f1 --complement)
      name=$(echo $entry | sed 's/^\s*//' | cut -d ' ' -f1)
      case $type in
        "2")  # directory
          all_dirs=$(echo "$all_dirs $dir/$name" | sed 's/^\s*//')
          ;;
        *)
          (if [ "$partition_name" != "system" ]; then printf %s "/$partition_name"; fi; echo "$dir/$name" | sed 's#^//#/#') >> "$file_list_file"
          ;;
      esac
    done <<< "$entries"
  done
  sort -n -o "$file_list_file" "$file_list_file"

  # Diff
  grep "FileName: /${partition_name}/" $product_out/sbom.spdx | sed 's/^FileName: //' | sort -n > "$files_in_spdx_file"
  diff "$file_list_file" "$files_in_spdx_file"
done