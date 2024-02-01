#!/bin/bash -eu
#
# Copyright 2017 Google Inc. All rights reserved.
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

# To track how long we took to startup.
case $(uname -s) in
  Darwin)
    export TRACE_BEGIN_SOONG=`$T/prebuilts/build-tools/path/darwin-x86/date +%s%3N`
    ;;
  *)
    export TRACE_BEGIN_SOONG=$(date +%s%N)
    ;;
esac

source $(cd $(dirname $BASH_SOURCE) &> /dev/null && pwd)/../make/shell_utils.sh
require_top

# Save the current PWD for use in soong_ui
export ORIGINAL_PWD=${PWD}
export TOP=$(gettop)
source ${TOP}/build/soong/scripts/microfactory.bash

soong_build_go soong_ui android/soong/cmd/soong_ui
soong_build_go mk2rbc android/soong/mk2rbc/mk2rbc
soong_build_go rbcrun rbcrun/rbcrun
soong_build_go release-config android/soong/cmd/release_config/release_config

new_args=()
do_thing=false
for arg in "$@"
do
  if [ "$arg" = 'general-tests' ] || [ "$arg" = 'mts' ] || [ "$arg" = 'acts_tests' ] || [ "$arg" = 'gts' ] || [ "$arg" = 'vts' ] || [ "$arg" = 'wts' ] || [ "$arg" = 'catbox' ] || [ "$arg" = 'gcatbox' ] || [ "$arg" = 'bluetooth_stack_with_facade' ] || [ "$arg" = 'haiku-presubmit' ] || [ "$arg" = 'wvts' ] || [ "$arg" = 'device-tests' ]
  then
    echo "skip"
  else
    new_args+=("$arg")
  fi
done

cd ${TOP}
exec "$(getoutdir)/soong_ui" "${new_args[@]}"
