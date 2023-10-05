#!/bin/bash -eu

set -o pipefail
source "$(dirname "$0")/../../bazel/lib.sh"

UUID="blank"
PROFILE_OUT="blank"
b_args=$(formulate_b_args "build --config=nonsense foo:bar")

if [[ $b_args != "build --profile=$PROFILE_OUT/bazel_metrics-profile --config=bp2build --invocation_id=$UUID --config=metrics_data --config=nonsense foo:bar" ]]; then
   echo "b args are malformed"
   echo "Expected : build --profile=$PROFILE_OUT/bazel_metrics-profile --config=bp2build  --invocation_id=$UUID --config=metrics_data --config=nonsense foo:bar"
   echo "Actual: $b_args"
   exit 1
fi

b_args=$(formulate_b_args "build --config=nonsense --disable_bes --package_path \"my package\" foo:bar")

if [[ $b_args != "build --profile=$PROFILE_OUT/bazel_metrics-profile --config=bp2build --invocation_id=$UUID --config=nonsense --package_path \"my package\" foo:bar" ]]; then
   echo "b args are malformed"
   echo "Expected : build --profile=$PROFILE_OUT/bazel_metrics-profile --config=bp2build  --invocation_id=$UUID --config=nonsense --package_path \"my package\" foo:bar"
   echo "Actual: $b_args"
   exit 1
fi
