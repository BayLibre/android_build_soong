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

new_args=()
do_thing=false
for arg in "$@"
do
  if [ "$arg" = 'general-tests' ]
  then
    # do stuff
    do_thing=true
    echo "skip general-tests"
  else
    new_args+=("$arg")
  fi
done

cd ${TOP}
echo "Dist dir:"
echo "${DIST_DIR}"

#printenv

#HOST_TESTCASES=$($(getoutdir)/soong_ui --dumpvar-mode --abs HOST_OUT_TESTCASES)
#TARGET_TESTCASES=$($(getoutdir)/soong_ui --dumpvar-mode --abs TARGET_OUT_TESTCASES)

if [ "$do_thing" = true ] ; then
  # merge these 2 lines later so you don't need to reinput the TARGET_* stuff
#  time "$(getoutdir)/soong_ui" --make-mode TARGET_PRODUCT=aosp_x86_64 TARGET_RELEASE=trunk_staging WITH_DEXPREOPT_BOOT_IMG_AND_SYSTEM_SERVER_ONLY=true cts-tradefed compatibility-host-util vts-tradefed hello_world_test StsCommonUtilTests ats-tradefed-tests vts_kernel_ltp_tests soong_zip
"$(getoutdir)/soong_ui" "${new_args[@]}" cts-tradefed compatibility-host-util vts-tradefed hello_world_test StsCommonUtilTests ats-tradefed-tests vts_kernel_ltp_tests soong_zip
time "${TOP}/out/host/linux-x86/bin/soong_zip" -C ${TOP}/out/host/linux-x86 -P host/ -D ${TOP}/out/host/linux-x86/testcases/ats-tradefed-tests -D ${TOP}/out/host/linux-x86/testcases/hello_world_test -D ${TOP}/out/host/linux-x86/testcases/StsCommonUtilTests -D ${TOP}/out/host/linux-x86/testcases/vts_kernel_ltp_tests -C ${TOP}/out/target/product/generic_x86_64 -P target/testcases -D ${TOP}/out/target/product/generic_x86_64/testcases/hello_world_test -C ${TOP}/out/host/linux-x86/framework -P host/tools/ -f ${TOP}/out/host/linux-x86/framework/cts-tradefed.jar -f ${TOP}/out/host/linux-x86/framework/compatibility-host-util.jar -f ${TOP}/out/host/linux-x86/framework/vts-tradefed.jar -o ${TOP}/out/target/product/generic_x86_64/general-tests-2.zip
  exit 0
fi

cd ${TOP}
exec "$(getoutdir)/soong_ui" "$@"
