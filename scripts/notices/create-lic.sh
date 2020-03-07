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

# This tool creates tab-separated license references for targets.

set -u

ME=$(basename $0)
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

TAB=$(echo | tr '\n' '\t')

target=
alias=
licenses=

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
            licenses="${licenses} ${1}"
            ;;
        esac;;
    esac
    shift
done

for license in ${licenses}; do
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
    project=$(dirname "${license}")"/"
    while [ "${project}" != './' ]; do
        if [ -d "${project}/.git/" ]; then
            break
        fi
        project=$(dirname "${project}")
        if [ "${project}" == '/' ] then
            # Terminate loop if absolute path reaches root.
            break
        fi
        project="${project}/"
    done
    # Rest of the path inside project
    if [ "${project}" == './' ]; then
        rest="${license}"
    else
        rest="${license:${#project}}"
    fi
    # If hardware, vendor, device or kernel appear in subdirectory name within
    # the project, obscure all but basename.
    case "${rest}" in
      *hardware*)
        echo "${prefix}hardware/${fname}"
        continue
        ;;
      *vendor*)
        echo "${prefix}vendor/${fname}"
        continue
        ;;
      *device*)
        echo "${prefix}device/${fname}"
        continue
        ;;
      *kernel*)
        echo "${prefix}kernel/${fname}"
        continue
        ;;
    esac
    # If hardware, vendor, device or kernel appear in project name, obscure
    # project name only preserving the subdirectory name within the project.
    case "${project}" in
      *hardware*)
        echo "${prefix}hardware/${rest}"
        continue
        ;;
      *vendor*)
        echo "${prefix}vendor/${rest}"
        continue
        ;;
      *device*)
        echo "${prefix}device/${rest}"
        continue
        ;;
      *kernel*)
        echo "${prefix}kernel/${rest}"
        continue
        ;;
    esac
    # Nothing to obscure, report the entire path to the license.
    echo "${prefix}${license}"
done
