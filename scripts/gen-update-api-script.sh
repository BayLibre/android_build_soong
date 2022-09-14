#!/bin/bash -eu

# Copyright 2022 Google Inc. All rights reserved.
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


# This is a helper script for the update-api family of build targets
# This script will generate a script to copy the updated api files from out/ back to the source tree

# The generated script will be run outside ninja execution
# This indirection is necessary since the source tree will be made read-only during ninja execution

function parse_args(){
  options=$(getopt -l "current_api_gen:,current_api_src:,removed_api_gen:,removed_api_src:,out:" -o "" -- "$@")

  eval set -- "$options"
  while true; do
    case $1 in
      --current_api_gen)
        shift;
        current_api_gen=$1
        ;;
      --current_api_src)
        shift;
        current_api_src=$1
        ;;
      --removed_api_gen)
        shift;
        removed_api_gen=$1
        ;;
      --removed_api_src)
        shift;
        removed_api_src=$1
        ;;
      --out)
        shift;
        out=$1
        ;;
      --)
        shift;
        break;
    esac
    shift
  done
}
parse_args $@

# Create file
# Copy .txt file if contents have changed
# Selectively copying the file ensures that depedendent targets like `checkapi` do not become dirty after each i`m update-api` run, even when no APIs have been added/removed
cat <<EOF > ${out}
#!/bin/bash -eu
if ! cmp -s ${current_api_gen} ${current_api_src}; then cp -f ${current_api_gen} ${current_api_src}; fi
if ! cmp -s ${removed_api_gen} ${removed_api_src}; then cp -f ${removed_api_gen} ${removed_api_src}; fi
EOF


# Make file executable
chmod +x ${out}
