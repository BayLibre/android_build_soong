#!/bin/bash
# A manual script to rerun `m` multiple times to ensure null builds for successive builds
if [[ -z $TARGET_PRODUCT ]] ; then
  echo "lunch ?"
  exit 1
fi

readonly target=$1
if [[ -z $target ]] ; then
  echo "USAGE: verify_null_builds.sh <target> [<loop_count>]"
  echo "  target:     e.g. nothing, libc, droid etc"
  echo "  loop_count: how many times to run the build, default is 4"
  exit 1
fi

readonly loop_count=$(if [[ -z $2 ]] ; then echo 4; else echo "$2"; fi)
readonly base_dir=runs
mkdir -p $base_dir
run_id=$(find $base_dir -mindepth 1 -maxdepth 1 | sed -e "s/^$base_dir\\/.\+-\([0-9]\+\)/\\1/g" | sort -rn |  head -n 1)
run_id=$(( run_id + 1 ))
readonly report_root="$base_dir/$target-$run_id"
mkdir "$report_root"
source build/envsetup.sh

function run_m {
  local -r loop=$1
  local -r ninja_start=$(if [[ -f out/.ninja_log ]]; then echo $(( 1 + $(wc -l < out/.ninja_log) )); else echo 1; fi)
  local -r clean_or_incremental=$(if [[ -d out ]] ; then echo "incremental"; else echo "clean"; fi)
  local -r legacy_or_mixed=$(if [[ $USE_BAZEL_ANALYSIS -eq 1 ]] ; then echo "mixed"; else echo "legacy"; fi)
  local -r log_file="$report_root/$clean_or_incremental-$legacy_or_mixed-$loop.log"
  echo -e "\033[1;45m************ Loop $loop see $log_file **********************************\033[0m"
  if [[ -f $log_file ]] ; then
    echo "$log_file already exists"
    return 1
  fi
  if ! NINJA_STATUS="[%f/%t] " NINJA_ARGS="-d explain -d keepdepfile" \
    m "$target" --skip-soong-tests >"$log_file"; then
      echo "FAILED"
      return 1
  else
    tail "$log_file" | grep "build completed"
    local -r new_ninja=$(tail -n "+$ninja_start" out/.ninja_log | wc -l)
    local -r globbings=$(tail -n "+$ninja_start" out/.ninja_log | grep -c "out/soong/globs/build/[0-9]\+")
    echo "𝝙 .ninja_log: $new_ninja lines (globbing: $globbings)"
    local -r explain_lines=$(grep -c "^ninja explain:" "$log_file")
    local -r phony_outputs=$(grep -c "^ninja explain: edge with output .\+ is a phony output, so is always dirty" "$log_file")
    echo "     explain: $explain_lines lines (phony: $phony_outputs)"
    if [[ $clean_or_incremental = "incremental" && $explain_lines -ne $phony_outputs ]]; then
      echo -e "\033[1;31mNULL BUILD EXPECTED\033[0m"
      exit 1
    fi
  fi
}

for ((i=1; i <= loop_count; i++)); do
  run_m "$i"
done

