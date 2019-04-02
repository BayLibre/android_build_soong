#!/bin/bash
#
# Copyright (C) 2019 The Android Open Source Project
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

if [[ $# -le 1 ]]; then
  cat <<EOF
Usage:
  package-check.sh <jar-file> <package-list>
Checks that the class files in the <jar file> are in the <package-list> or
sub-packages.
EOF
  exit 1
fi

JAR_FILE=$1
shift
if [[ ! -f ${JAR_FILE} ]]; then
  echo "jar file \"${JAR_FILE}\" does not exist."
  exit 1
fi

PREFIXES=()
while [[ $# -ge 1 ]]; do
  PACKAGE="$1"
  if [[ "${PACKAGE//\//}" != "${PACKAGE}" ]]; then
    echo "Invalid package \"${PACKAGE}\". Use dot notation for packages."
    exit 1
  fi
  # Transform to a slash-separated path and add a trailing slash to enforce
  # package name boundary.
  PREFIXES+=("${PACKAGE//\./\/}/")
  shift
done

# Get the file names from the jar file.
ZIP_CONTENTS=`zipinfo -1 $JAR_FILE`
if [[ $? -ne 0 ]]; then
  echo "zipinfo failed"
  exit 1
fi

# Check all class file names against the expected prefixes.
CLASS_SUFFIX='.class'
OLD_IFS=${IFS}
IFS=$'\n'
for ZIP_ENTRY in ${ZIP_CONTENTS}; do
  # Check the suffix.
  if [[ ${#ZIP_ENTRY} -ge ${#CLASS_SUFFIX} && \
        "${ZIP_ENTRY:$((${#ZIP_ENTRY}-${#CLASS_SUFFIX}))}" == \
            "${CLASS_SUFFIX}" ]]; then
    # Match against prefixes.
    FOUND=false
    for PREFIX in ${PREFIXES[@]}; do
      if [[ "${ZIP_ENTRY:0:${#PREFIX}}" = "${PREFIX}" ]]; then
        FOUND=true
        break
      fi
    done
    if [[ "${FOUND}" == "false" ]]; then
      echo "Class file ${ZIP_ENTRY} is outside specified packages."
      exit 1
    fi
  fi
done
IFS=${OLD_IFS}
