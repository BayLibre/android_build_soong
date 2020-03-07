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

# This tool creates tab-separated license references for targets.

set -eu

ME=$(basename "$0")
USAGE="usage: ${ME} [-a projectAlias] <target> <license or notice>...

Create a tab-separated license record for 'target' with a separate line for
each license or notice file that exists.

If -a is specified, the path information for each license or notice is replaced
in the external name with the given project alias.

When not specified, the path information will be replaced with 'proprietary' or
'private' whenever the path contains 'proprietary' or 'private' respectively.
If the text 'hardware', 'vendor', 'device' or 'kernel' appears in the relative
path within the project, the entire path information will be replaced with
'hardware', 'vendor', 'device' or 'kernel'. If the same text appears in the
project name, the external name will preserve the path within the project, but
will simplify the project name to 'hardware', 'vendor', 'device' or 'kernel'
respectively.

Each line consists of the following tab-separated fields:
  1. The target name.
  2. The path to the original license file.
  3. The external name of the license file. (may be aliased)
"

# If 'hardware', 'vendor', 'device', or 'kernel' appear in 1st argument,
# obscure path by prepending the matched text to the 2nd argument.
#
# Returns true if obscured.
function obscured() {
  case "${1}" in
      *hardware*)
        echo "${prefix}hardware/${2}"
        ;;
      *vendor*)
        echo "${prefix}vendor/${2}"
        ;;
      *device*)
        echo "${prefix}device/${2}"
        ;;
      *kernel*)
        echo "${prefix}kernel/${2}"
        ;;
       *)
        return 1
        ;;
    esac
    return 0
}

TAB=$'\t'

target=
alias=
declare -a licenses

while [ $# -gt 0 ]; do
    case "${1-}" in
      -a=*|--a=*)
        alias=$(expr "${1}" : '--?a=\(.*\)$';;
      -a)
        alias="${2?"${USAGE}"}"; shift;;
      *)
        case "${target}" in
          '')
            target="${1}"
            ;;
          *)
            licenses+=(${1})
            ;;
        esac;;
    esac
    shift
done

for idx in "${!licenses[*]}"; do
    license="${licenses[${idx}]"
    if ! [ -f "${license}" ]; then
      # Nothing to do if license doesn't exist as a file.
      continue
    fi
    # Base filename
    fname=$(basename "${license}")
    # Always output the same raw prefix of target and full path.
    prefix="${target}${TAB}${license}${TAB}"
    # If alias given, always obscure path using alias.
    if [ -n "${alias}" ]; then
        echo "${prefix}${alias}/${fname}"
        continue
    fi
    # If 'proprietary' or 'private' appear anywhere, obscure all but basename.
    case "${license}" in
      *proprietary*)
        echo "${prefix}proprietary/${fname}"
        continue
        ;;
      *private*)
        echo "${prefix}private/${fname}"
        continue
        ;;
    esac
    # Find the git project name.
    project="$(dirname "${license}")/"
    while [ "${project}" != './' ] &&
          [ ! -d "${project}/.git" ] &&
          [ "${project}" != "/" ]; do
      project="$(dirname "${project}")/"
    done
    # Rest of the path inside project
    if [ "${project}" == './' ] || [ "${project}" == '//' ]; then
        rest="${license}"
    else
        rest="${license:${#project}}"
    fi
    # If hardware, vendor, device or kernel appear in subdirectory name within
    # the project, obscure all but basename.
    if obscured "${rest}" "${fname}"; then
        continue
    fi
    # If hardware, vendor, device or kernel appear in project name, obscure
    # project name only preserving the subdirectory name within the project.
    if obscured "${project}" "${rest}"; then
        continue
    fi
    # Nothing to obscure, report the entire path to the license.
    echo "${prefix}${license}"
done
