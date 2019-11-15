#!/bin/bash -e

PREBUILT_APEX_OUT_DIR=${PREBUILT_APEX_OUT_DIR:-out-apex}

if [ "${OUT_DIR}" = "${PREBUILT_APEX_OUT_DIR}" ]; then
  # get_build_var runs Soong in --dumpvar-mode, which will create various files
  # in ${OUT_DIR} based on the empty soong.variables. When we replace it below
  # these other files will affect Soong and produce a different
  # ${OUT_DIR}/soong/build.ninja.
  echo "OUT_DIR must not be '${PREBUILT_APEX_OUT_DIR}' while build variables are queried."
  exit 1
fi

source build/envsetup.sh

PLATFORM_SDK_VERSION=$(get_build_var PLATFORM_SDK_VERSION)
PLATFORM_VERSION=$(get_build_var PLATFORM_VERSION)
PLATFORM_VERSION_ALL_CODENAMES=$(get_build_var PLATFORM_VERSION_ALL_CODENAMES)

# PLATFORM_VERSION_ALL_CODENAMES is a comma separated list like O,P. We need to
# turn this into ["O","P"].
PLATFORM_VERSION_ALL_CODENAMES=${PLATFORM_VERSION_ALL_CODENAMES/,/'","'}
PLATFORM_VERSION_ALL_CODENAMES="[\"${PLATFORM_VERSION_ALL_CODENAMES}\"]"

export OUT_DIR=${PREBUILT_APEX_OUT_DIR}

# Logic from build/make/core/goma.mk
if [ "${USE_GOMA}" = true ]; then
  if [ -n "${GOMA_DIR}" ]; then
    goma_dir="${GOMA_DIR}"
  else
    goma_dir="${HOME}/goma"
  fi
  GOMA_CC="${goma_dir}/gomacc"
  export CC_WRAPPER="${CC_WRAPPER}${CC_WRAPPER:+ }${GOMA_CC}"
  export CXX_WRAPPER="${CXX_WRAPPER}${CXX_WRAPPER:+ }${GOMA_CC}"
  export JAVAC_WRAPPER="${JAVAC_WRAPPER}${JAVAC_WRAPPER:+ }${GOMA_CC}"
else
  USE_GOMA=false
fi

SOONG_OUT=${OUT_DIR}/soong
mkdir -p ${SOONG_OUT}
SOONG_VARS=${SOONG_OUT}/soong.variables

# We only really need to set some of these variables, but soong won't merge this
# with the defaults, so we need to write out all the defaults with our values
# added.
cat > ${SOONG_VARS}.new << EOF
{
    "Platform_sdk_version": ${PLATFORM_SDK_VERSION},
    "Platform_sdk_codename": "${PLATFORM_VERSION}",
    "Platform_version_active_codenames": ${PLATFORM_VERSION_ALL_CODENAMES},

    "DeviceName": "generic_arm64",
    "HostArch": "x86_64",
    "Prebuilt_apex_abis": true,

    "UseGoma": ${USE_GOMA}
}
EOF

if [ -f ${SOONG_VARS} ] && cmp -s ${SOONG_VARS} ${SOONG_VARS}.new; then
  # Don't touch soong.variables if we don't have to, to avoid Soong rebuilding
  # the ninja file when it isn't necessary.
  rm ${SOONG_VARS}.new
else
  mv ${SOONG_VARS}.new ${SOONG_VARS}
fi

m --skip-make "$@"
