#!/bin/bash -eu

set -o pipefail

# This test exercises the bootstrapping process of the build system
# in a source tree that only contains enough files for Bazel and Soong to work.

source "$(dirname "$0")/lib.sh"

readonly GENERATED_BUILD_FILE_NAME="BUILD.bazel"

function test_android_bp() {
  setup
  mkdir -p a
  pwd
  cat > a/foo.rs <<'EOF'
        fn main() {}
EOF
  cat > a/Android.bp <<'EOF'
        rust_library {
                    name: "libfizz_buzz",
                    crate_name:"fizz_buzz",
                    srcs: ["foo.rs"],
        }
        rust_binary {
                    name: "fizz-buzz",
                    crate_name:"fizz-buzz-bin",
                    srcs: ["foo.rs"],
                    dylibs: ["libfizz_buzz"],
        }
EOF
  run_ninja fizz-buzz
}

test_android_bp
