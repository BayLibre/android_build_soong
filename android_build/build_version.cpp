#include <android-build.h>

namespace android::build {

BuildType type() {
#if defined(BUILD_TYPE_DEBUGGABLE)
  #if defined(BUILD_TYPE_ENG)
    return kBuildTypeEng;
  #else
    return kBuildTypeUserDebug;
  #endif
#else
  return kBuildTypeUser;
#endif
}

namespace version {
  const char* incremental() {
    return "filled in via objcopy or something?";
  }
}

}  // namespace android::build
