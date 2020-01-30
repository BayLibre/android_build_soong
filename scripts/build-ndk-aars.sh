#!/bin/bash -ex

# Copyright 2019 Google Inc. All rights reserved.
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

export OUT_DIR=${OUT_DIR:-out}

if [ -e ${OUT_DIR}/soong/.soong.in_make ]; then
  # If ${OUT_DIR} has been created without --skip-make, Soong will create an
  # ${OUT_DIR}/soong/build.ninja that leaves out many targets which are
  # expected to be supplied by the .mk files, and that might cause errors in
  # "m --skip-make" below. We therefore default to a different out dir
  # location in that case.
  AAR_OUT_DIR=out-aar
  echo "Avoiding in-make OUT_DIR '${OUT_DIR}' - building in '${AAR_OUT_DIR}' instead"
  OUT_DIR=${AAR_OUT_DIR}
fi

TOP=$(pwd)

source build/envsetup.sh

my_get_build_var() {
  # get_build_var will run Soong in normal in-make mode where it creates
  # .soong.in_make. That would clobber our real out directory, so we need to
  # run it in a different one.
  OUT_DIR=${OUT_DIR}/get_build_var get_build_var "$@"
}

PLATFORM_SDK_VERSION=$(my_get_build_var PLATFORM_SDK_VERSION)
PLATFORM_VERSION_CODENAME=$(my_get_build_var PLATFORM_VERSION_CODENAME)
PLATFORM_VERSION_ALL_CODENAMES=$(my_get_build_var PLATFORM_VERSION_ALL_CODENAMES)
PLATFORM_VERSION=$(my_get_build_var PLATFORM_VERSION)
APPS_DEFAULT_VERSION_NAME=$(my_get_build_var APPS_DEFAULT_VERSION_NAME)

if [ "$PLATFORM_VERSION_CODENAME" == "REL" ]; then
  PLATFORM_SDK_FINAL=true
else
  PLATFORM_SDK_FINAL=false
fi

# PLATFORM_VERSION_ALL_CODENAMES is a comma separated list like O,P. We need to
# turn this into ["O","P"].
PLATFORM_VERSION_ALL_CODENAMES=${PLATFORM_VERSION_ALL_CODENAMES/,/'","'}
PLATFORM_VERSION_ALL_CODENAMES="[\"${PLATFORM_VERSION_ALL_CODENAMES}\"]"

SOONG_OUT=${OUT_DIR}/soong
mkdir -p ${SOONG_OUT}
SOONG_VARS=${SOONG_OUT}/soong.variables

# We only really need to set some of these variables, but soong won't merge this
# with the defaults, so we need to write out all the defaults with our values
# added.
cat > ${SOONG_VARS}.new << EOF
{
    "Ndk_abis": true,
    "Exclude_draft_ndk_apis": true,

    "Platform_version_name": "${PLATFORM_VERSION}",
    "Platform_sdk_version": ${PLATFORM_SDK_VERSION},
    "Platform_sdk_codename": "${PLATFORM_VERSION_CODENAME}",
    "Platform_sdk_final": ${PLATFORM_SDK_FINAL},
    "Platform_version_active_codenames": ${PLATFORM_VERSION_ALL_CODENAMES},
    "DeviceName": "generic_arm64",
    "HostArch": "x86_64",
    "AppsDefaultVersionName": "${APPS_DEFAULT_VERSION_NAME}"
}
EOF

if [ -f ${SOONG_VARS} ] && cmp -s ${SOONG_VARS} ${SOONG_VARS}.new; then
  # Don't touch soong.variables if we don't have to, to avoid Soong rebuilding
  # the ninja file when it isn't necessary.
  rm ${SOONG_VARS}.new
else
  mv ${SOONG_VARS}.new ${SOONG_VARS}
fi

INTERMEDIATES=${SOONG_OUT}/.intermediates
NDK_AARS=(
  libnativehelper/libnativehelper_aar/android_common/libnativehelper_aar.aar
)

TARGETS=("${NDK_AARS[@]/#/$INTERMEDIATES/}")

m --skip-make $TARGETS

if [ -n "${DIST_DIR}" ]; then
    mkdir -p ${DIST_DIR} || true
    for aar in ${TARGETS}; do
      mv $aar ${DIST_DIR}
    done
fi
