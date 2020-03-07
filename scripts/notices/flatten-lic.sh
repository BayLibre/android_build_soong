#!/bin/bash

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

# This tool combines--under a new target--tab-separated license references from
# its dependencies.

set -eu

ME=$(basename "$0")
USAGE="usage: ${ME} <output file> <target> <tab-separated license file>...

Combines .lic.tab files replacing the target and removing duplicates.
Allows overwriting one of the input files with the output. (useful with xargs)

e.g.
  ${ME} myprog.lic.tab myprog myprog.o.lic.tab mylib.a.lic.tab

After making 'myprog' to distribute out of 'myprog.o' and 'mylib.a', the user
does not see the .o or the .a separately. It makes sense to re-associate the
licenses with 'myprog', which the user does see.
"

ofile="${1?"${USAGE}"}"
shift

target="${1?"${USAGE}"}"
shift

if [ $# -lt 1 ]; then
  # no input, so output empty file
  : >"${ofile}"
  exit $?
fi

# Separate calculating the output from overwriting the output file in case the
# same file appears as an input.
output=$(
    awk -v FS='\t' -v OFS='\t' -v target="${target}" \
        '{ $1 = target; print }' "$*" | sort -u
)
echo "${output}" >"${ofile}"
