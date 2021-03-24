#!/bin/bash -eu

# This test exercises the bootstrapping process of the build system
# in a source tree that only contains enough files for Bazel and Soong to work.

source build/soong/test_framework.sh

HARDWIRED_MOCK_TOP=
# Uncomment this to be able to view the source tree after a test is run
# HARDWIRED_MOCK_TOP=/tmp/td

export GOCACHE=$(mktemp -t -d gc.XXXXXX)
trap 'cd / && rm -fr "$GOCACHE"' EXIT

function setup() {
  create_mock_top

  copy_directory build/blueprint
  copy_directory build/soong

  symlink_directory prebuilts/go
  symlink_directory prebuilts/build-tools
  symlink_directory external/golang-protobuf

  touch "$MOCK_TOP/Android.bp"

  export ALLOW_MISSING_DEPENDENCIES=true

  mkdir -p out/soong
}

function run_soong() {
  build/soong/soong_ui.bash --make-mode --skip-ninja --skip-make --skip-soong-tests
}

function test_noop {
  echo "NOOP"
}

function test_smoke {
  setup
  run_soong
}

function test_null_build() {
  setup
  run_soong
  local bootstrap_mtime1=$(stat -c "%y" out/soong/.bootstrap/build.ninja)
  local output_mtime1=$(stat -c "%y" out/soong/build.ninja)
  run_soong
  local bootstrap_mtime2=$(stat -c "%y" out/soong/.bootstrap/build.ninja)
  local output_mtime2=$(stat -c "%y" out/soong/build.ninja)

  if [[ "$bootstrap_mtime1" == "$bootstrap_mtime2" ]]; then
    # Bootstrapping is always done. It doesn't take a measurable amount of time.
    fail "Bootstrap Ninja file did not change on null build"
  fi

  if [[ "$output_mtime1" != "$output_mtime2" ]]; then
    fail "Output Ninja file changed on null build"
  fi
}

function test_soong_build_rebuilt_if_blueprint_changes() {
  setup
  run_soong
  local mtime1=$(stat -c "%y" out/soong/.bootstrap/build.ninja)

  sed -i 's/pluginGenSrcCmd/pluginGenSrcCmd2/g' build/blueprint/bootstrap/bootstrap.go

  run_soong
  local mtime2=$(stat -c "%y" out/soong/.bootstrap/build.ninja)

  if [[ "$mtime1" == "$mtime2" ]]; then
    fail "Bootstrap Ninja file did not change"
  fi
}

function test_change_android_bp() {
  setup
  mkdir -p a
  cat > a/Android.bp <<'EOF'
python_binary_host {
  name: "my_little_binary_host",
  srcs: ["my_little_binary_host.py"]
}
EOF
  touch a/my_little_binary_host.py
  run_soong

  grep -q "^# Module:.*my_little_binary_host" out/soong/build.ninja || fail "module not found"

  cat > a/Android.bp <<'EOF'
python_binary_host {
  name: "my_great_binary_host",
  srcs: ["my_great_binary_host.py"]
}
EOF
  touch a/my_great_binary_host.py
  run_soong

  grep -q "^# Module:.*my_little_binary_host" out/soong/build.ninja && fail "old module found"
  grep -q "^# Module:.*my_great_binary_host" out/soong/build.ninja || fail "new module not found"
}


function test_add_android_bp() {
  setup
  run_soong
  local mtime1=$(stat -c "%y" out/soong/build.ninja)

  mkdir -p a
  cat > a/Android.bp <<'EOF'
python_binary_host {
  name: "my_little_binary_host",
  srcs: ["my_little_binary_host.py"]
}
EOF
  touch a/my_little_binary_host.py
  run_soong

  local mtime2=$(stat -c "%y" out/soong/build.ninja)
  if [[ "$mtime1" == "$mtime2" ]]; then
    fail "Output Ninja file did not change"
  fi

  grep -q "^# Module:.*my_little_binary_host$" out/soong/build.ninja || fail "New module not in output"

  run_soong
}

function test_delete_android_bp() {
  setup
  mkdir -p a
  cat > a/Android.bp <<'EOF'
python_binary_host {
  name: "my_little_binary_host",
  srcs: ["my_little_binary_host.py"]
}
EOF
  touch a/my_little_binary_host.py
  run_soong

  grep -q "^# Module:.*my_little_binary_host$" out/soong/build.ninja || fail "Module not in output"

  rm a/Android.bp
  run_soong

  grep -q "^# Module:.*my_little_binary_host$" out/soong/build.ninja && fail "Old module in output"
  return 0
}

function test_add_file_to_glob() {
  setup

  mkdir -p a
  cat > a/Android.bp <<'EOF'
python_binary_host {
  name: "my_little_binary_host",
  srcs: ["*.py"],
}
EOF
  touch a/my_little_binary_host.py
  run_soong
  local mtime1=$(stat -c "%y" out/soong/build.ninja)

  touch a/my_little_library.py
  run_soong

  local mtime2=$(stat -c "%y" out/soong/build.ninja)
  if [[ "$mtime1" == "$mtime2" ]]; then
    fail "Output Ninja file did not change"
  fi

  grep -q my_little_library.py out/soong/build.ninja || fail "new file is not in output"
}

function test_soong_build_rerun_iff_environment_changes() {
  setup

  mkdir -p cherry
  cat > cherry/Android.bp <<'EOF'
bootstrap_go_package {
  name: "cherry",
  pkgPath: "android/soong/cherry",
  deps: [
    "blueprint",
    "soong",
    "soong-android",
  ],
  srcs: [
    "cherry.go",
  ],
  pluginFor: ["soong_build"],
}
EOF

  cat > cherry/cherry.go <<'EOF'
package cherry

import (
  "android/soong/android"
  "github.com/google/blueprint"
)

var (
  pctx = android.NewPackageContext("cherry")
)

func init() {
  android.RegisterSingletonType("cherry", CherrySingleton)
}

func CherrySingleton() android.Singleton {
  return &cherrySingleton{}
}

type cherrySingleton struct{}

func (p *cherrySingleton) GenerateBuildActions(ctx android.SingletonContext) {
  cherryRule := ctx.Rule(pctx, "cherry",
    blueprint.RuleParams{
      Command: "echo CHERRY IS " + ctx.Config().Getenv("CHERRY") + " > ${out}",
      CommandDeps: []string{},
      Description: "Cherry",
    })

  outputFile := android.PathForOutput(ctx, "cherry", "cherry.txt")
  var deps android.Paths

  ctx.Build(pctx, android.BuildParams{
    Rule: cherryRule,
    Output: outputFile,
    Inputs: deps,
  })
}
EOF

  export CHERRY=TASTY
  run_soong
  grep -q "CHERRY IS TASTY" out/soong/build.ninja \
    || fail "first value of environment variable is not used"

  export CHERRY=RED
  run_soong
  grep -q "CHERRY IS RED" out/soong/build.ninja \
    || fail "second value of environment variable not used"
  local mtime1=$(stat -c "%y" out/soong/build.ninja)

  run_soong
  local mtime2=$(stat -c "%y" out/soong/build.ninja)
  if [[ "$mtime1" != "$mtime2" ]]; then
    fail "Output Ninja file changed when environment variable did not"
  fi

}

function test_add_file_to_soong_build() {
  setup
  run_soong
  local mtime1=$(stat -c "%y" out/soong/build.ninja)

  mkdir -p a
  cat > a/Android.bp <<'EOF'
bootstrap_go_package {
  name: "picard-soong-rules",
  pkgPath: "android/soong/picard",
  deps: [
    "blueprint",
    "soong",
    "soong-android",
  ],
  srcs: [
    "picard.go",
  ],
  pluginFor: ["soong_build"],
}
EOF

  cat > a/picard.go <<'EOF'
package picard

import (
  "android/soong/android"
  "github.com/google/blueprint"
)

var (
  pctx = android.NewPackageContext("picard")
)

func init() {
  android.RegisterSingletonType("picard", PicardSingleton)
}

func PicardSingleton() android.Singleton {
  return &picardSingleton{}
}

type picardSingleton struct{}

func (p *picardSingleton) GenerateBuildActions(ctx android.SingletonContext) {
  picardRule := ctx.Rule(pctx, "picard",
    blueprint.RuleParams{
      Command: "echo Make it so. > ${out}",
      CommandDeps: []string{},
      Description: "Something quotable",
    })

  outputFile := android.PathForOutput(ctx, "picard", "picard.txt")
  var deps android.Paths

  ctx.Build(pctx, android.BuildParams{
    Rule: picardRule,
    Output: outputFile,
    Inputs: deps,
  })
}

EOF

  run_soong
  local mtime2=$(stat -c "%y" out/soong/build.ninja)
  if [[ "$mtime1" == "$mtime2" ]]; then
    fail "Output Ninja file did not change"
  fi

  grep -q "Make it so" out/soong/build.ninja || fail "New action not present"
}

function test_null_build_after_docs {
  setup
  run_soong
  local mtime1=$(stat -c "%y" out/soong/build.ninja)

  prebuilts/build-tools/linux-x86/bin/ninja -f out/soong/build.ninja soong_docs
  run_soong
  local mtime2=$(stat -c "%y" out/soong/build.ninja)

  if [[ "$mtime1" != "$mtime2" ]]; then
    fail "Output Ninja file changed on null build"
  fi
}

run_test_suite
