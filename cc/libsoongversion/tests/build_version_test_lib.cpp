#include <build/version.h>

#include "build_version_test_lib.h"

std::string __attribute__((visibility("default"))) LibGetBuildNumber() {
  return android::build::GetBuildNumber();
}
