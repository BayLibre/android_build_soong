#pragma once

namespace android {
namespace build {

enum BuildType {
  kBuildTypeUser,
  kBuildTypeUserDebug,
  kBuildTypeEng,
};

BuildType type();
const char* product();
const char* id();

namespace version {

const char* incremental();
unsigned int sdk_version();

}  // namespace version

}  // namespace build
}  // namespace android
