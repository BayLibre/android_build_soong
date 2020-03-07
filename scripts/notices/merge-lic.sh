#!/bin/sh

# Copyright 2020 Google Inc. All rights reserved.
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

# This tool combines tab-separated target to license mappings.

set -u

ME=$(basename $0)
USAGE="usage: ${ME} <output file> <tab-separated license file>...

Combines multiple .lic.tab files removing duplicates, and allows overwriting
one of the input files with the output, if necessary. (useful with xargs)

e.g.
  ${ME} etc/NOTICE.lic.tab bin/myprog.lic.tab lib/mylib.so.lic.tab

or

  # start with empty file
  : >NOTICE_FILES/NOTICE.lic.tab
  # use the same file as the output and as the 1st input of (possibly) many
  find NOTICE_FILES/ -type f -print0 | \
      xargs -0 ${ME} NOTICE_FILES/NOTICE.lic.tab NOTICE_FILES/NOTICE.lic.tab
"

ofile="${1?"${USAGE}"}"
shift

if [ $# -lt 1 ]; then
  # no input, so output an empty file
  : >"${ofile}"
  exit $?
fi

output=$(awk -v FS='\t' -v OFS='\t' '{print}' "$*" | sort -u)
echo "${output}" >"${ofile}"
