#!/bin/bash -e

cd "$(dirname "${BASH_SOURCE[0]}")"/../../..

source build/envsetup.sh
lunch aosp_arm64-eng
sed -i '/exception_perf/d' out/.ninja_log
ANDROID_QUIET_BUILD=true mmm build/soong/exception_perf

echo

grep exception_perf out/.ninja_log | grep "test.o" | \
  awk '{ printf("%s %.2fs\n", $4, ($2-$1)/1000) }' | \
  tr '/' ' ' | \
  awk '{ print $8 " " $7 " " $14 }' | \
  sort | \
  column -t
