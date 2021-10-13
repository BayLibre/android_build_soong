#!/bin/bash -eu

set -o pipefail

# Tests of converting license functionality of the androidmk tool

$ANDROID_BUILD_TOP/build/soong/soong_ui.bash --make-mode androidmk

source "$(dirname "$0")/lib.sh"

# Expect to create a new license module
function test_rewrite_license_property_inside_current_directory {
  setup

  # Create an Android.mk file
  mkdir -p a/b
  cat > a/b/Android.mk <<'EOF'
include $(CLEAR_VARS)
LOCAL_MODULE := foo
LOCAL_LICENSE_KINDS := license_kind1 license_kind2
LOCAL_LICENSE_CONDITIONS := license_condition
LOCAL_NOTICE_FILE := $(LOCAL_PATH)/license_notice1 $(LOCAL_PATH)/license_notice2
include $(BUILD_PACKAGE)
EOF

  # Create an expected Android.bp file for the module "foo"
  cat > a/b/Android.bp <<'EOF'
package {
    // See: http://go/android-license-faq
    default_applicable_licenses: [
        "a_b_license",
    ],
}

license {
    name: "a_b_license",
    visibility: [":__subpackages__"],
    license_kinds: [
        "license_kind1",
        "license_kind2",
    ],
    license_text: [
        "license_notice1",
        "license_notice2",
    ],
}

android_app {
    name: "foo",
}
EOF

  ANDROID_BUILD_TOP="$MOCK_TOP"
  local out=$(androidmk a/b/Android.mk)
  local expected=$(<a/b/Android.bp)

  if [[ "$out" != "$expected" ]]; then
    cleanup_mock_top
    fail "The output is not the same as the expected"
  fi

  cleanup_mock_top
}

# Expect to reference to an existing license module
function test_rewrite_license_property_outside_current_directory {
  setup

  # Create an Android.mk file
  mkdir -p a/b/c/d
  cat > a/b/c/d/Android.mk <<'EOF'
include $(CLEAR_VARS)
LOCAL_MODULE := foo
LOCAL_LICENSE_KINDS := license_kind1 license_kind2
LOCAL_LICENSE_CONDITIONS := license_condition
LOCAL_NOTICE_FILE := $(LOCAL_PATH)/../../license_notice1 $(LOCAL_PATH)/../../license_notice2
include $(BUILD_PACKAGE)
EOF

  # Create an expected Android.bp file for the module "foo"
  cat > a/b/c/d/Android.bp <<'EOF'
package {
    // See: http://go/android-license-faq
    default_applicable_licenses: [
        "a_b_license",
    ],
}

android_app {
    name: "foo",
}
EOF

  # Create an expected Android.bp file at a/b/
  cat > a/b/Android.bp <<'EOF'
package {
    // See: http://go/android-license-faq
    default_applicable_licenses: [
        "a_b_license",
    ],
}

license {
    name: "a_b_license",
    visibility: [":__subpackages__"],
    license_kinds: [
        "license_kind1",
        "license_kind2",
    ],
    license_text: [
        "license_notice1",
        "license_notice2",
    ],
}

android_app {
    name: "bar",
}
EOF

  ANDROID_BUILD_TOP="$MOCK_TOP"
  local out=$(androidmk a/b/c/d/Android.mk)
  local expected=$(<a/b/c/d/Android.bp)

  if [[ "$out" != "$expected" ]]; then
    cleanup_mock_top
    fail "The output is not the same as the expected"
  fi

  cleanup_mock_top
}

test_rewrite_license_property_inside_current_directory

test_rewrite_license_property_outside_current_directory
