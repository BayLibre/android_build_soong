#include <android-build.h>

extern "C" char __android_build_version_incremental[128] = "PLACEHOLDER";

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
    return __android_build_version_incremental;
  }
}

}  // namespace android::build
