#!/bin/bash -eu

REAL_TOP="$(readlink -f "$(dirname "$0")"/../..)"

function fail {
  echo ERROR: $1
  exit 1
}

function copy_directory() {
  local dir="$1"
  local parent="$(dirname "$dir")"

  mkdir -p "$MOCK_TOP/$parent"
  cp -R "$REAL_TOP/$dir" "$MOCK_TOP/$parent"
}

function symlink_directory() {
  local dir="$1"

  mkdir -p "$MOCK_TOP/$dir"
  for i in $(ls "$REAL_TOP/$dir"); do
    local target="$MOCK_TOP/$dir/$i"
    local source="$REAL_TOP/$dir/$i"

    if [[ -e "$target" ]]; then
      if [[ ! -d "$source" || ! -d "$target" ]]; then
        fail "Trying to symlink $dir twice"
      fi
    else
      ln -s "$REAL_TOP/$dir/$i" "$MOCK_TOP/$dir/$i";
    fi
  done
}

function create_and_cd_mock_source_tree() {
  #MOCK_TOP=/tmp/td
  #rm -fr /tmp/td
  #mkdir -p /tmp/td

  MOCK_TOP=$(mktemp -t -d st.XXXXX)
  trap 'echo cd / && echo rm -fr "$MOCK_TOP"' EXIT
  echo "Test mock client: $MOCK_TOP"
  cd "$MOCK_TOP"

  copy_directory build/blueprint
  copy_directory build/soong

  symlink_directory prebuilts/go
  symlink_directory prebuilts/build-tools
  symlink_directory external/golang-protobuf

  touch "$MOCK_TOP/Android.bp"

  export ALLOW_MISSING_DEPENDENCIES=true

  mkdir -p out/soong
  # This is necessary because the empty soong.variables file written to satisfy
  # Ninja would contain "BootJars: {}" instead of "BootJars: []" which cannot
  # be parsed back
  cat > out/soong/soong.variables <<'EOF'
{
    "BuildNumberFile": "build_number.txt",
    "Platform_version_name": "S",
    "Platform_sdk_version": 30,
    "Platform_sdk_codename": "S",
    "Platform_sdk_final": false,
    "Platform_version_active_codenames": [
        "S"
    ],
    "Platform_vndk_version": "S",
    "DeviceName": "generic_arm64",
    "DeviceArch": "arm64",
    "DeviceArchVariant": "armv8-a",
    "DeviceCpuVariant": "generic",
    "DeviceAbi": [
        "arm64-v8a"
    ],
    "DeviceSecondaryArch": "arm",
    "DeviceSecondaryArchVariant": "armv8-a",
    "DeviceSecondaryCpuVariant": "generic",
    "DeviceSecondaryAbi": [
        "armeabi-v7a",
        "armeabi"
    ],
    "HostArch": "x86_64",
    "HostSecondaryArch": "x86",
    "CrossHost": "windows",
    "CrossHostArch": "x86",
    "CrossHostSecondaryArch": "x86_64",
    "AAPTCharacteristics": "nosdcard",
    "AAPTConfig": [
        "normal",
        "large",
        "xlarge",
        "hdpi",
        "xhdpi",
        "xxhdpi"
    ],
    "AAPTPreferredConfig": "xhdpi",
    "AAPTPrebuiltDPI": [
        "xhdpi",
        "xxhdpi"
    ],
    "Malloc_not_svelte": true,
    "Malloc_zero_contents": true,
    "Malloc_pattern_fill_contents": false,
    "Safestack": false,
    "BootJars": [],
    "UpdatableBootJars": [],
    "Native_coverage": null
}
EOF
}

function test_bootstrap_build() {
  create_and_cd_mock_source_tree
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
}

function test_null_build() {
  create_and_cd_mock_source_tree
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
  MTIME1=$(stat -c "%y" out/soong/build.ninja)
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
  MTIME2=$(stat -c "%y" out/soong/build.ninja)

  if [[ "$MTIME1" != "$MTIME2" ]]; then
    fail "Null build was not a null build"
  fi
}

function test_null_build_bootstrap_ninja_unchanged() {
  create_and_cd_mock_source_tree
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
  MTIME1=$(stat -c "%y" out/soong/.bootstrap/build.ninja)
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
  MTIME2=$(stat -c "%y" out/soong/.bootstrap/build.ninja)

  if [[ "$MTIME1" != "$MTIME2" ]]; then
    fail "Bootstrap Ninja file changed on null build"
  fi
}

function test_soong_build_rebuilt_if_blueprint_changes() {
  create_and_cd_mock_source_tree
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
  MTIME1=$(stat -c "%y" out/soong/.bootstrap/build.ninja)

  sed -i 's/pluginGenSrcCmd/pluginGenSrcCmd2/g' build/blueprint/bootstrap/bootstrap.go

  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
  MTIME2=$(stat -c "%y" out/soong/.bootstrap/build.ninja)

  if [[ "$MTIME1" == "$MTIME2" ]]; then
    fail "Bootstrap Ninja file did not change"
  fi
}

test_null_build_bootstrap_ninja_unchanged
test_soong_build_rebuilt_if_blueprint_changes
#test_bootstrap_build
