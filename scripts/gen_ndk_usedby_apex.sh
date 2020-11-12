#!/bin/bash -e

# Copyright 2018 Google Inc. All rights reserved.
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

# Generates NDK API txt file used by Mainline modules.
parseReadelfOutput() {
  while IFS= read -r line
  do
      if [[ $line = *FUNC*GLOBAL*UND*@* ]] ;
      then
          echo "$line" | sed -r 's/.*UND (.*)@.*/\1/g' >> "$2"
      fi
  done < "$1"
  echo "" >> "$2"
}

unzipJarAndApk() {
  tmpUnzippedDir="$1"/tmpUnzipped
  [[ -e "$tmpUnzippedDir" ]] && rm -rf "$tmpUnzippedDir"
  mkdir -p "$tmpUnzippedDir"
  find "$1" -name "*.jar" -exec unzip -o {} -d "$tmpUnzippedDir" \;
  find "$1" -name "*.apk" -exec unzip -o {} -d "$tmpUnzippedDir" \;
  find "$tmpUnzippedDir" -name "*.MF" -exec rm {} \;
}

lookForExecFile() {
  dir="$1"
  readelf="$2"
  find "$dir" -type f -name "*.so"  -exec "$2" --dyn-symbols {} >> "$dir"/../tmpReadelf.txt \;
  find "$dir" -type f -perm /111 ! -name "*.so"  -exec "$2" --dyn-symbols {} >> "$dir"/../tmpReadelf.txt \;
}

unzipJarAndApk "$2"
lookForExecFile "$2" "$3"
tmpReadelfOutput="$2/../tmpReadelf.txt"
[[ -e "$4" ]] && rm "$4"
parseReadelfOutput "$tmpReadelfOutput" "$4"
[[ -e "$tmpReadelfOutput" ]] && rm "$tmpReadelfOutput"
rm -rf "$2/tmpUnzipped"
