// Copyright 2022 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package allowlists

// Configuration to decide if modules in a directory should default to true/false for bp2build_available
type Bp2BuildConfig map[string]BazelConversionConfigEntry
type BazelConversionConfigEntry int

const (
	// iota + 1 ensures that the int value is not 0 when used in the Bp2buildAllowlist map,
	// which can also mean that the key doesn't exist in a lookup.

	// all modules in this package and subpackages default to bp2build_available: true.
	// allows modules to opt-out.
	Bp2BuildDefaultTrueRecursively BazelConversionConfigEntry = iota + 1

	// all modules in this package (not recursively) default to bp2build_available: true.
	// allows modules to opt-out.
	Bp2BuildDefaultTrue

	// all modules in this package (not recursively) default to bp2build_available: false.
	// allows modules to opt-in.
	Bp2BuildDefaultFalse
)

var (
	Bp2buildDefaultConfig = Bp2BuildConfig{
		"art/libartpalette":                     Bp2BuildDefaultTrueRecursively,
		"art/libartbase":                        Bp2BuildDefaultTrueRecursively,
		"art/libdexfile":                        Bp2BuildDefaultTrueRecursively,
		"art/libnativebridge":                   Bp2BuildDefaultTrueRecursively,
		"art/runtime":                           Bp2BuildDefaultTrueRecursively,
		"art/tools":                             Bp2BuildDefaultTrue,
		"bionic":                                Bp2BuildDefaultTrueRecursively,
		"bootable/recovery/tools/recovery_l10n": Bp2BuildDefaultTrue,

		"build/bazel/examples/aidl":                   Bp2BuildDefaultTrueRecursively,
		"build/bazel/examples/apex/minimal":           Bp2BuildDefaultTrueRecursively,
		"build/bazel/examples/soong_config_variables": Bp2BuildDefaultTrueRecursively,
		"build/bazel/examples/python":                 Bp2BuildDefaultTrueRecursively,
		"build/bazel/examples/gensrcs":                Bp2BuildDefaultTrueRecursively,
		"build/make/target/product/security":          Bp2BuildDefaultTrue,
		"build/make/tools/signapk":                    Bp2BuildDefaultTrue,
		"build/make/tools/zipalign":                   Bp2BuildDefaultTrueRecursively,
		"build/soong":                                 Bp2BuildDefaultTrue,
		"build/soong/cc/libbuildversion":              Bp2BuildDefaultTrue, // Skip tests subdir
		"build/soong/cc/ndkstubgen":                   Bp2BuildDefaultTrue,
		"build/soong/cc/symbolfile":                   Bp2BuildDefaultTrue,
		"build/soong/linkerconfig":                    Bp2BuildDefaultTrueRecursively,
		"build/soong/scripts":                         Bp2BuildDefaultTrueRecursively,

		"cts/common/device-side/nativetesthelper/jni": Bp2BuildDefaultTrueRecursively,
		"development/apps/DevelopmentSettings":        Bp2BuildDefaultTrue,
		"development/apps/Fallback":                   Bp2BuildDefaultTrue,
		"development/apps/WidgetPreview":              Bp2BuildDefaultTrue,
		"development/samples/BasicGLSurfaceView":      Bp2BuildDefaultTrue,
		"development/samples/BluetoothChat":           Bp2BuildDefaultTrue,
		"development/samples/BrokenKeyDerivation":     Bp2BuildDefaultTrue,
		"development/samples/Compass":                 Bp2BuildDefaultTrue,
		"development/samples/ContactManager":          Bp2BuildDefaultTrue,
		"development/samples/FixedGridLayout":         Bp2BuildDefaultTrue,
		"development/samples/HelloEffects":            Bp2BuildDefaultTrue,
		"development/samples/Home":                    Bp2BuildDefaultTrue,
		"development/samples/HoneycombGallery":        Bp2BuildDefaultTrue,
		"development/samples/JetBoy":                  Bp2BuildDefaultTrue,
		"development/samples/KeyChainDemo":            Bp2BuildDefaultTrue,
		"development/samples/LceDemo":                 Bp2BuildDefaultTrue,
		"development/samples/LunarLander":             Bp2BuildDefaultTrue,
		"development/samples/MultiResolution":         Bp2BuildDefaultTrue,
		"development/samples/MultiWindow":             Bp2BuildDefaultTrue,
		"development/samples/NotePad":                 Bp2BuildDefaultTrue,
		"development/samples/Obb":                     Bp2BuildDefaultTrue,
		"development/samples/RSSReader":               Bp2BuildDefaultTrue,
		"development/samples/ReceiveShareDemo":        Bp2BuildDefaultTrue,
		"development/samples/SearchableDictionary":    Bp2BuildDefaultTrue,
		"development/samples/SipDemo":                 Bp2BuildDefaultTrue,
		"development/samples/SkeletonApp":             Bp2BuildDefaultTrue,
		"development/samples/Snake":                   Bp2BuildDefaultTrue,
		"development/samples/SpellChecker/":           Bp2BuildDefaultTrueRecursively,
		"development/samples/ThemedNavBarKeyboard":    Bp2BuildDefaultTrue,
		"development/samples/ToyVpn":                  Bp2BuildDefaultTrue,
		"development/samples/TtsEngine":               Bp2BuildDefaultTrue,
		"development/samples/USB/AdbTest":             Bp2BuildDefaultTrue,
		"development/samples/USB/MissileLauncher":     Bp2BuildDefaultTrue,
		"development/samples/VoiceRecognitionService": Bp2BuildDefaultTrue,
		"development/samples/VoicemailProviderDemo":   Bp2BuildDefaultTrue,
		"development/samples/WiFiDirectDemo":          Bp2BuildDefaultTrue,
		"development/sdk":                             Bp2BuildDefaultTrueRecursively,

		"external/aac":                           Bp2BuildDefaultTrueRecursively,
		"external/arm-optimized-routines":        Bp2BuildDefaultTrueRecursively,
		"external/auto/android-annotation-stubs": Bp2BuildDefaultTrueRecursively,
		"external/auto/common":                   Bp2BuildDefaultTrueRecursively,
		"external/auto/service":                  Bp2BuildDefaultTrueRecursively,
		"external/boringssl":                     Bp2BuildDefaultTrueRecursively,
		"external/bouncycastle":                  Bp2BuildDefaultTrue,
		"external/brotli":                        Bp2BuildDefaultTrue,
		"external/conscrypt":                     Bp2BuildDefaultTrue,
		"external/e2fsprogs":                     Bp2BuildDefaultTrueRecursively,
		"external/eigen":                         Bp2BuildDefaultTrueRecursively,
		"external/erofs-utils":                   Bp2BuildDefaultTrueRecursively,
		"external/error_prone":                   Bp2BuildDefaultTrueRecursively,
		"external/expat":                         Bp2BuildDefaultTrueRecursively,
		"external/f2fs-tools":                    Bp2BuildDefaultTrue,
		"external/flac":                          Bp2BuildDefaultTrueRecursively,
		"external/fmtlib":                        Bp2BuildDefaultTrueRecursively,
		"external/google-benchmark":              Bp2BuildDefaultTrueRecursively,
		"external/googletest":                    Bp2BuildDefaultTrueRecursively,
		"external/gwp_asan":                      Bp2BuildDefaultTrueRecursively,
		"external/hamcrest":                      Bp2BuildDefaultTrueRecursively,
		"external/icu":                           Bp2BuildDefaultTrueRecursively,
		"external/icu/android_icu4j":             Bp2BuildDefaultFalse, // java rules incomplete
		"external/icu/icu4j":                     Bp2BuildDefaultFalse, // java rules incomplete
		"external/jarjar":                        Bp2BuildDefaultTrueRecursively,
		"external/javapoet":                      Bp2BuildDefaultTrueRecursively,
		"external/jemalloc_new":                  Bp2BuildDefaultTrueRecursively,
		"external/jsoncpp":                       Bp2BuildDefaultTrueRecursively,
		"external/junit":                         Bp2BuildDefaultTrueRecursively,
		"external/libavc":                        Bp2BuildDefaultTrueRecursively,
		"external/libcap":                        Bp2BuildDefaultTrueRecursively,
		"external/libcxx":                        Bp2BuildDefaultTrueRecursively,
		"external/libcxxabi":                     Bp2BuildDefaultTrueRecursively,
		"external/libevent":                      Bp2BuildDefaultTrueRecursively,
		"external/libgav1":                       Bp2BuildDefaultTrueRecursively,
		"external/libhevc":                       Bp2BuildDefaultTrueRecursively,
		"external/libjpeg-turbo":                 Bp2BuildDefaultTrueRecursively,
		"external/libmpeg2":                      Bp2BuildDefaultTrueRecursively,
		"external/libpng":                        Bp2BuildDefaultTrueRecursively,
		"external/libyuv":                        Bp2BuildDefaultTrueRecursively,
		"external/lz4/lib":                       Bp2BuildDefaultTrue,
		"external/lzma/C":                        Bp2BuildDefaultTrueRecursively,
		"external/mdnsresponder":                 Bp2BuildDefaultTrueRecursively,
		"external/minijail":                      Bp2BuildDefaultTrueRecursively,
		"external/pcre":                          Bp2BuildDefaultTrueRecursively,
		"external/protobuf":                      Bp2BuildDefaultTrueRecursively,
		"external/python/six":                    Bp2BuildDefaultTrueRecursively,
		"external/rappor":                        Bp2BuildDefaultTrueRecursively,
		"external/scudo":                         Bp2BuildDefaultTrueRecursively,
		"external/selinux/libselinux":            Bp2BuildDefaultTrueRecursively,
		"external/selinux/libsepol":              Bp2BuildDefaultTrueRecursively,
		"external/speex":                         Bp2BuildDefaultTrueRecursively,
		"external/toybox":                        Bp2BuildDefaultTrueRecursively,
		"external/zlib":                          Bp2BuildDefaultTrueRecursively,
		"external/zopfli":                        Bp2BuildDefaultTrueRecursively,
		"external/zstd":                          Bp2BuildDefaultTrueRecursively,

		"frameworks/av/media/codecs":                         Bp2BuildDefaultTrueRecursively,
		"frameworks/av/media/liberror":                       Bp2BuildDefaultTrueRecursively,
		"frameworks/av/services/minijail":                    Bp2BuildDefaultTrueRecursively,
		"frameworks/base/media/tests/MediaDump":              Bp2BuildDefaultTrue,
		"frameworks/base/services/tests/servicestests/aidl":  Bp2BuildDefaultTrue,
		"frameworks/base/startop/apps/test":                  Bp2BuildDefaultTrue,
		"frameworks/base/tests/appwidgets/AppWidgetHostTest": Bp2BuildDefaultTrueRecursively,
		"frameworks/native/libs/adbd_auth":                   Bp2BuildDefaultTrueRecursively,
		"frameworks/native/libs/arect":                       Bp2BuildDefaultTrueRecursively,
		"frameworks/native/libs/math":                        Bp2BuildDefaultTrueRecursively,
		"frameworks/native/libs/nativebase":                  Bp2BuildDefaultTrueRecursively,
		"frameworks/native/opengl/tests/gl2_cameraeye":       Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/gl2_java":            Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/testLatency":         Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/testPauseResume":     Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/testViewport":        Bp2BuildDefaultTrue,
		"frameworks/proto_logging/stats/stats_log_api_gen":   Bp2BuildDefaultTrueRecursively,

		"libnativehelper":                                  Bp2BuildDefaultTrueRecursively,
		"packages/apps/DevCamera":                          Bp2BuildDefaultTrue,
		"packages/apps/HTMLViewer":                         Bp2BuildDefaultTrue,
		"packages/apps/Protips":                            Bp2BuildDefaultTrue,
		"packages/apps/WallpaperPicker":                    Bp2BuildDefaultTrue,
		"packages/modules/StatsD/lib/libstatssocket":       Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb":                             Bp2BuildDefaultTrue,
		"packages/modules/adb/apex":                        Bp2BuildDefaultTrue,
		"packages/modules/adb/crypto":                      Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/libs":                        Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/pairing_auth":                Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/pairing_connection":          Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/proto":                       Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/tls":                         Bp2BuildDefaultTrueRecursively,
		"packages/providers/MediaProvider/tools/dialogs":   Bp2BuildDefaultFalse, // TODO(b/242834374)
		"packages/screensavers/Basic":                      Bp2BuildDefaultTrue,
		"packages/services/Car/tests/SampleRearViewCamera": Bp2BuildDefaultFalse, // TODO(b/242834321)

		"prebuilts/clang/host/linux-x86":           Bp2BuildDefaultTrueRecursively,
		"prebuilts/runtime/mainline/platform/sdk":  Bp2BuildDefaultTrueRecursively,
		"prebuilts/sdk/current/extras/app-toolkit": Bp2BuildDefaultTrue,
		"prebuilts/sdk/current/support":            Bp2BuildDefaultTrue,
		"prebuilts/tools/common/m2":                Bp2BuildDefaultTrue,

		"system/apex":                                            Bp2BuildDefaultFalse, // TODO(b/207466993): flaky failures
		"system/apex/apexer":                                     Bp2BuildDefaultTrue,
		"system/apex/libs":                                       Bp2BuildDefaultTrueRecursively,
		"system/apex/proto":                                      Bp2BuildDefaultTrueRecursively,
		"system/apex/tools":                                      Bp2BuildDefaultTrueRecursively,
		"system/core/debuggerd":                                  Bp2BuildDefaultTrueRecursively,
		"system/core/diagnose_usb":                               Bp2BuildDefaultTrueRecursively,
		"system/core/libasyncio":                                 Bp2BuildDefaultTrue,
		"system/core/libcrypto_utils":                            Bp2BuildDefaultTrueRecursively,
		"system/core/libcutils":                                  Bp2BuildDefaultTrueRecursively,
		"system/core/libpackagelistparser":                       Bp2BuildDefaultTrueRecursively,
		"system/core/libprocessgroup":                            Bp2BuildDefaultTrue,
		"system/core/libprocessgroup/cgrouprc":                   Bp2BuildDefaultTrue,
		"system/core/libprocessgroup/cgrouprc_format":            Bp2BuildDefaultTrue,
		"system/core/libsystem":                                  Bp2BuildDefaultTrueRecursively,
		"system/core/libutils":                                   Bp2BuildDefaultTrueRecursively,
		"system/core/libvndksupport":                             Bp2BuildDefaultTrueRecursively,
		"system/core/property_service/libpropertyinfoparser":     Bp2BuildDefaultTrueRecursively,
		"system/core/property_service/libpropertyinfoserializer": Bp2BuildDefaultTrueRecursively,
		"system/libartpalette":                                   Bp2BuildDefaultTrueRecursively,
		"system/libbase":                                         Bp2BuildDefaultTrueRecursively,
		"system/libfmq":                                          Bp2BuildDefaultTrue,
		"system/libhidl/transport/base/1.0":                      Bp2BuildDefaultTrue,
		"system/libhidl/transport/manager/1.0":                   Bp2BuildDefaultTrue,
		"system/libhidl/transport/manager/1.1":                   Bp2BuildDefaultTrue,
		"system/libhidl/transport/manager/1.2":                   Bp2BuildDefaultTrue,
		"system/libhwbinder":                                     Bp2BuildDefaultTrueRecursively,
		"system/libprocinfo":                                     Bp2BuildDefaultTrue,
		"system/libziparchive":                                   Bp2BuildDefaultTrueRecursively,
		"system/logging/liblog":                                  Bp2BuildDefaultTrueRecursively,
		"system/media/audio":                                     Bp2BuildDefaultTrueRecursively,
		"system/media/audio_utils":                               Bp2BuildDefaultTrueRecursively,
		"system/memory/libion":                                   Bp2BuildDefaultTrueRecursively,
		"system/memory/libmemunreachable":                        Bp2BuildDefaultTrueRecursively,
		"system/sepolicy/apex":                                   Bp2BuildDefaultTrueRecursively,
		"system/timezone/apex":                                   Bp2BuildDefaultTrueRecursively,
		"system/timezone/output_data":                            Bp2BuildDefaultTrueRecursively,
		"system/tools/sysprop":                                   Bp2BuildDefaultTrue,
		"system/unwinding/libunwindstack":                        Bp2BuildDefaultTrueRecursively,

		"tools/apksig": Bp2BuildDefaultTrue,
		"tools/platform-compat/java/android/compat":  Bp2BuildDefaultTrueRecursively,
		"tools/tradefederation/prebuilts/filegroups": Bp2BuildDefaultTrueRecursively,
	}

	Bp2buildKeepExistingBuildFile = map[string]bool{
		// This is actually build/bazel/build.BAZEL symlinked to ./BUILD
		".":/*recursive = */ false,

		// build/bazel/examples/apex/... BUILD files should be generated, so
		// build/bazel is not recursive. Instead list each subdirectory under
		// build/bazel explicitly.
		"build/bazel":/* recursive = */ false,
		"build/bazel/ci/dist":/* recursive = */ false,
		"build/bazel/examples/android_app":/* recursive = */ true,
		"build/bazel/examples/cc":/* recursive = */ true,
		"build/bazel/examples/java":/* recursive = */ true,
		"build/bazel/examples/partitions":/* recursive = */ true,
		"build/bazel/bazel_skylib":/* recursive = */ true,
		"build/bazel/rules":/* recursive = */ true,
		"build/bazel/rules_cc":/* recursive = */ true,
		"build/bazel/scripts":/* recursive = */ true,
		"build/bazel/tests":/* recursive = */ true,
		"build/bazel/platforms":/* recursive = */ true,
		"build/bazel/product_config":/* recursive = */ true,
		"build/bazel/product_variables":/* recursive = */ true,
		"build/bazel/vendor/google":/* recursive = */ true,
		"build/bazel_common_rules":/* recursive = */ true,
		// build/make/tools/signapk BUILD file is generated, so build/make/tools is not recursive.
		"build/make/tools":/* recursive = */ false,
		"build/pesto":/* recursive = */ true,
		"build/soong/ui/metrics/bp2build_progress_metrics_proto":/* recursive = */ true,

		// external/bazelbuild-rules_android/... is needed by mixed builds, otherwise mixed builds analysis fails
		// e.g. ERROR: Analysis of target '@soong_injection//mixed_builds:buildroot' failed
		"external/bazelbuild-rules_android":/* recursive = */ true,
		"external/bazel-skylib":/* recursive = */ true,
		"external/guava":/* recursive = */ true,
		"external/jsr305":/* recursive = */ true,
		"frameworks/ex/common":/* recursive = */ true,

		"packages/apps/Music":/* recursive = */ true,
		"packages/apps/QuickSearchBox":/* recursive = */ true,

		"prebuilts/bazel":/* recursive = */ true,
		"prebuilts/bundletool":/* recursive = */ true,
		"prebuilts/gcc":/* recursive = */ true,
		"prebuilts/build-tools":/* recursive = */ true,
		"prebuilts/jdk/jdk11":/* recursive = */ false,
		"prebuilts/misc":/* recursive = */ false, // not recursive because we need bp2build converted build files in prebuilts/misc/common/asm
		"prebuilts/sdk":/* recursive = */ false,
		"prebuilts/sdk/tools":/* recursive = */ false,
		"prebuilts/r8":/* recursive = */ false,
	}

	Bp2buildModuleAlwaysConvertList = map[string]struct{}{
		// cc mainline modules
		"code_coverage.policy":                       struct{}{},
		"code_coverage.policy.other":                 struct{}{},
		"codec2_soft_exports":                        struct{}{},
		"com.android.media.swcodec-androidManifest":  struct{}{},
		"com.android.media.swcodec-ld.config.txt":    struct{}{},
		"com.android.media.swcodec-mediaswcodec.rc":  struct{}{},
		"com.android.media.swcodec.certificate":      struct{}{},
		"com.android.media.swcodec.key":              struct{}{},
		"com.android.neuralnetworks-androidManifest": struct{}{},
		"com.android.neuralnetworks.certificate":     struct{}{},
		"com.android.neuralnetworks.key":             struct{}{},
		"flatbuffer_headers":                         struct{}{},
		"gemmlowp_headers":                           struct{}{},
		"gl_headers":                                 struct{}{},
		"libandroid_runtime_lazy":                    struct{}{},
		"libandroid_runtime_vm_headers":              struct{}{},
		"libaudioclient_aidl_conversion_util":        struct{}{},
		"libbinder":                                  struct{}{},
		"libbinder_device_interface_sources":         struct{}{},
		"libbinder_aidl":                             struct{}{},
		"libbinder_headers":                          struct{}{},
		"libbinder_headers_platform_shared":          struct{}{},
		"libbluetooth-types-header":                  struct{}{},
		"libbufferhub_headers":                       struct{}{},
		"libcodec2":                                  struct{}{},
		"libcodec2_headers":                          struct{}{},
		"libcodec2_internal":                         struct{}{},
		"libdmabufheap":                              struct{}{},
		"libdvr_headers":                             struct{}{},
		"libgsm":                                     struct{}{},
		"libgui_bufferqueue_sources":                 struct{}{},
		"libhardware":                                struct{}{},
		"libhardware_headers":                        struct{}{},
		"libincfs_headers":                           struct{}{},
		"libnativeloader-headers":                    struct{}{},
		"libnativewindow_headers":                    struct{}{},
		"libneuralnetworks_headers":                  struct{}{},
		"libopus":                                    struct{}{},
		"libpdx_headers":                             struct{}{},
		"libprocpartition":                           struct{}{},
		"libruy_static":                              struct{}{},
		"libandroidio":                               struct{}{},
		"libandroidio_srcs":                          struct{}{},
		"libserviceutils":                            struct{}{},
		"libstagefright_enc_common":                  struct{}{},
		"libstagefright_foundation_headers":          struct{}{},
		"libstagefright_headers":                     struct{}{},
		"libsurfaceflinger_headers":                  struct{}{},
		"libsync":                                    struct{}{},
		"libtextclassifier_hash_headers":             struct{}{},
		"libtextclassifier_hash_static":              struct{}{},
		"libtflite_kernel_utils":                     struct{}{},
		"libtinyxml2":                                struct{}{},
		"libui-types":                                struct{}{},
		"libui_headers":                              struct{}{},
		"libvorbisidec":                              struct{}{},
		"media_ndk_headers":                          struct{}{},
		"media_plugin_headers":                       struct{}{},
		"mediaswcodec.policy":                        struct{}{},
		"mediaswcodec.xml":                           struct{}{},
		"philox_random":                              struct{}{},
		"philox_random_headers":                      struct{}{},
		"server_configurable_flags":                  struct{}{},
		"tensorflow_headers":                         struct{}{},

		// fastboot
		"bootimg_headers":             struct{}{},
		"fastboot":                    struct{}{},
		"libfastboot":                 struct{}{},
		"liblp":                       struct{}{},
		"libstorage_literals_headers": struct{}{},

		//external/avb
		"avbtool":     struct{}{},
		"libavb":      struct{}{},
		"avb_headers": struct{}{},

		//external/fec
		"libfec_rs": struct{}{},

		//system/core/libsparse
		"libsparse": struct{}{},

		//system/extras/ext4_utils
		"libext4_utils": struct{}{},
		"mke2fs_conf":   struct{}{},

		//system/extras/libfec
		"libfec": struct{}{},

		//system/extras/squashfs_utils
		"libsquashfs_utils": struct{}{},

		//system/extras/verity/fec
		"fec": struct{}{},

		//packages/apps/Car/libs/car-ui-lib/car-ui-androidx
		// genrule dependencies for java_imports
		"car-ui-androidx-annotation-nodeps":              struct{}{},
		"car-ui-androidx-collection-nodeps":              struct{}{},
		"car-ui-androidx-core-common-nodeps":             struct{}{},
		"car-ui-androidx-lifecycle-common-nodeps":        struct{}{},
		"car-ui-androidx-constraintlayout-solver-nodeps": struct{}{},

		//system/libhidl
		// needed by cc_hidl_library
		"libhidlbase": struct{}{},

		//frameworks/native
		"framework_native_aidl_binder": struct{}{},
		"framework_native_aidl_gui":    struct{}{},

		//frameworks/native/libs/input
		"inputconstants_aidl": struct{}{},
	}

	Bp2buildModuleTypeAlwaysConvertList = map[string]struct{}{
		"linker_config":          struct{}{},
		"java_import":            struct{}{},
		"java_import_host":       struct{}{},
		"aidl_interface_headers": struct{}{},
	}

	Bp2buildModuleDoNotConvertList = map[string]struct{}{
		// cc bugs
		"libactivitymanager_aidl":  struct{}{}, // TODO(b/207426160): Unsupported use of aidl sources (via Dactivity_manager_procstate_aidl) in a cc_library
		"gen-kotlin-build-file.py": struct{}{}, // TODO(b/198619163) module has same name as source

		// TODO(b/201816222): Requires sdk_version support.
		"libgtest_ndk_c++":      struct{}{},
		"libgtest_main_ndk_c++": struct{}{},

		// TODO(b/202876379): has arch-variant static_executable
		"linkerconfig": struct{}{},
		"mdnsd":        struct{}{},

		"linker":                 struct{}{}, // TODO(b/228316882): cc_binary uses link_crt
		"libdebuggerd":           struct{}{}, // TODO(b/228314770): support product variable-specific header_libs
		"versioner":              struct{}{}, // TODO(b/228313961):  depends on prebuilt shared library libclang-cpp_host as a shared library, which does not supply expected providers for a shared library
		"libvpx":                 struct{}{}, // TODO(b/240756936): Arm neon variant not supported
		"art_libartbase_headers": struct{}{}, // TODO(b/236268577): Header libraries do not support export_shared_libs_headers
		"apexer_test":            struct{}{}, // Requires aapt2
		"apexer_test_host_tools": struct{}{},
		"host_apex_verifier":     struct{}{},
		"tjbench":                struct{}{}, // TODO(b/240563612): Stem property

		// java bugs
		"libbase_ndk": struct{}{}, // TODO(b/186826477): fails to link libctscamera2_jni for device (required for CtsCameraTestCases)

		// python protos
		"libprotobuf-python": struct{}{}, // Has a handcrafted alternative

		// genrule incompatibilities
		"brotli-fuzzer-corpus": struct{}{}, // TODO(b/202015218): outputs are in location incompatible with bazel genrule handling.

		// TODO(b/203369847): multiple genrules in the same package creating the same file
		"platform_tools_properties":     struct{}{},
		"build_tools_source_properties": struct{}{},

		// aar support
		"prebuilt_car-ui-androidx-core-common":         struct{}{}, // TODO(b/224773339): genrule dependency creates an .aar, not a .jar
		"prebuilt_platform-robolectric-4.4-prebuilt":   struct{}{}, // aosp/1999250, needs .aar support in Jars
		"prebuilt_platform-robolectric-4.5.1-prebuilt": struct{}{}, // aosp/1999250, needs .aar support in Jars

		// proto support
		"libstats_proto_host": struct{}{}, // TODO(b/236055697): handle protos from other packages

		// path property for filegroups
		// TODO(b/210751803): we don't handle path property for filegroups
		"conscrypt":                        struct{}{},
		"conscrypt-for-host":               struct{}{},
		"host-libprotobuf-java-full":       struct{}{},
		"libprotobuf-internal-protos":      struct{}{},
		"libprotobuf-internal-python-srcs": struct{}{},
		"libprotobuf-java-full":            struct{}{},
		"libprotobuf-java-util-full":       struct{}{},
		"auto_value_plugin_resources":      struct{}{},

		// go deps:
		"analyze_bcpf":                       struct{}{}, // depends on bpmodify a blueprint_go_binary.
		"apex-protos":                        struct{}{}, // depends on soong_zip, a go binary
		"generated_android_icu4j_src_files":  struct{}{}, // depends on unconverted modules: soong_zip
		"generated_android_icu4j_test_files": struct{}{}, // depends on unconverted modules: soong_zip
		"icu4c_test_data":                    struct{}{}, // depends on unconverted modules: soong_zip

		"host_bionic_linker_asm":                struct{}{}, // depends on extract_linker, a go binary.
		"host_bionic_linker_script":             struct{}{}, // depends on extract_linker, a go binary.
		"libc_musl_sysroot_bionic_arch_headers": struct{}{}, // depends on soong_zip
		"libc_musl_sysroot_zlib_headers":        struct{}{}, // depends on soong_zip and zip2zip
		"libc_musl_sysroot_bionic_headers":      struct{}{}, // 218405924, depends on soong_zip and generates duplicate srcs
		"libc_musl_sysroot_libc++_headers":      struct{}{}, // depends on soong_zip, zip2zip
		"libc_musl_sysroot_libc++abi_headers":   struct{}{}, // depends on soong_zip, zip2zip
		"robolectric-sqlite4java-native":        struct{}{}, // depends on soong_zip, a go binary
		"robolectric_tzdata":                    struct{}{}, // depends on soong_zip, a go binary

		// rust support
		// rust conversions are not supported
		"libtombstoned_client_rust_bridge_code": struct{}{},
		"libtombstoned_client_wrapper":          struct{}{},

		// unconverted deps
		"CarHTMLViewer":                          struct{}{}, // depends on unconverted modules android.car-stubs, car-ui-lib
		"abb":                                    struct{}{}, // depends on unconverted modules: libcmd, libbinder
		"adb":                                    struct{}{}, // depends on unconverted modules: AdbWinApi, libandroidfw, libopenscreen-discovery, libopenscreen-platform-impl, libusb, bin2c_fastdeployagent, AdbWinUsbApi
		"android_icu4j_srcgen":                   struct{}{}, // depends on unconverted modules: currysrc
		"android_icu4j_srcgen_binary":            struct{}{}, // depends on unconverted modules: android_icu4j_srcgen, currysrc
		"apex_manifest_proto_java":               struct{}{}, // b/210751803, depends on libprotobuf-java-full
		"art-script":                             struct{}{}, // depends on unconverted modules: dalvikvm:, dex2oat
		"bin2c_fastdeployagent":                  struct{}{}, // depends on unconverted modules: deployagent
		"com.android.runtime":                    struct{}{}, // depends on unconverted modules: bionic-linker-config, linkerconfig
		"currysrc":                               struct{}{}, // depends on unconverted modules: currysrc_org.eclipse, guavalib, jopt-simple-4.9
		"dex2oat-script":                         struct{}{}, // depends on unconverted modules: dex2oat
		"generated_android_icu4j_resources":      struct{}{}, // depends on unconverted modules: android_icu4j_srcgen_binary, soong_zip
		"generated_android_icu4j_test_resources": struct{}{}, // depends on unconverted modules: android_icu4j_srcgen_binary, soong_zip
		"host-libprotobuf-java-nano":             struct{}{}, // b/220869005, depends on libprotobuf-java-nano
		"libadb_host":                            struct{}{}, // depends on unconverted modules: AdbWinApi, libopenscreen-discovery, libopenscreen-platform-impl, libusb
		"libapexutil":                            struct{}{}, // depends on unconverted modules: apex-info-list-tinyxml
		"libart":                                 struct{}{}, // depends on unconverted modules: apex-info-list-tinyxml, libtinyxml2, libnativeloader-headers, heapprofd_client_api, art_operator_srcs, libcpu_features, libodrstatslog, libelffile, art_cmdlineparser_headers, cpp-define-generator-definitions, libdexfile, libnativebridge, libnativeloader, libsigchain, libartbase, libprofile, cpp-define-generator-asm-support
		"libart-runtime-gtest":                   struct{}{}, // depends on unconverted modules: libgtest_isolated, libart-compiler, libdexfile, libprofile, libartbase, libartbase-art-gtest
		"libart_headers":                         struct{}{}, // depends on unconverted modules: art_libartbase_headers
		"libartbase-art-gtest":                   struct{}{}, // depends on unconverted modules: libgtest_isolated, libart, libart-compiler, libdexfile, libprofile
		"libartbased-art-gtest":                  struct{}{}, // depends on unconverted modules: libgtest_isolated, libartd, libartd-compiler, libdexfiled, libprofiled
		"libartd":                                struct{}{}, // depends on unconverted modules: art_operator_srcs, libcpu_features, libodrstatslog, libelffiled, art_cmdlineparser_headers, cpp-define-generator-definitions, libdexfiled, libnativebridge, libnativeloader, libsigchain, libartbased, libprofiled, cpp-define-generator-asm-support, apex-info-list-tinyxml, libtinyxml2, libnativeloader-headers, heapprofd_client_api
		"libartd-runtime-gtest":                  struct{}{}, // depends on unconverted modules: libgtest_isolated, libartd-compiler, libdexfiled, libprofiled, libartbased, libartbased-art-gtest
		"libdebuggerd_handler":                   struct{}{}, // depends on unconverted module libdebuggerd_handler_core
		"libdebuggerd_handler_core":              struct{}{}, // depends on unconverted module libdebuggerd
		"libdebuggerd_handler_fallback":          struct{}{}, // depends on unconverted module libdebuggerd
		"libdexfiled":                            struct{}{}, // depends on unconverted modules: dexfile_operator_srcs, libartbased, libartpalette
		"libfastdeploy_host":                     struct{}{}, // depends on unconverted modules: libandroidfw, libusb, AdbWinApi
		"libgmock_main_ndk":                      struct{}{}, // depends on unconverted modules: libgtest_ndk_c++
		"libgmock_ndk":                           struct{}{}, // depends on unconverted modules: libgtest_ndk_c++
		"libnativehelper_lazy_mts_jni":           struct{}{}, // depends on unconverted modules: libnativetesthelper_jni, libgmock_ndk
		"libnativehelper_mts_jni":                struct{}{}, // depends on unconverted modules: libnativetesthelper_jni, libgmock_ndk
		"libnativetesthelper_jni":                struct{}{}, // depends on unconverted modules: libgtest_ndk_c++
		"libprotobuf-java-nano":                  struct{}{}, // b/220869005, depends on non-public_current SDK
		"libstatslog":                            struct{}{}, // depends on unconverted modules: libstatspull, statsd-aidl-ndk, libbinder_ndk
		"libstatslog_art":                        struct{}{}, // depends on unconverted modules: statslog_art.cpp, statslog_art.h
		"linker_reloc_bench_main":                struct{}{}, // depends on unconverted modules: liblinker_reloc_bench_*
		"pbtombstone":                            struct{}{}, // depends on libdebuggerd, libunwindstack
		"crash_dump":                             struct{}{}, // depends on libdebuggerd, libunwindstack
		"robolectric-sqlite4java-0.282":          struct{}{}, // depends on unconverted modules: robolectric-sqlite4java-import, robolectric-sqlite4java-native
		"static_crasher":                         struct{}{}, // depends on unconverted modules: libdebuggerd_handler
		"stats-log-api-gen":                      struct{}{}, // depends on unconverted modules: libstats_proto_host
		"statslog.cpp":                           struct{}{}, // depends on unconverted modules: stats-log-api-gen
		"statslog.h":                             struct{}{}, // depends on unconverted modules: stats-log-api-gen
		"statslog.rs":                            struct{}{}, // depends on unconverted modules: stats-log-api-gen
		"statslog_art.cpp":                       struct{}{}, // depends on unconverted modules: stats-log-api-gen
		"statslog_art.h":                         struct{}{}, // depends on unconverted modules: stats-log-api-gen
		"statslog_header.rs":                     struct{}{}, // depends on unconverted modules: stats-log-api-gen
		"test_fips":                              struct{}{}, // depends on unconverted modules: adb
		"timezone-host":                          struct{}{}, // depends on unconverted modules: art.module.api.annotations
		"truth-host-prebuilt":                    struct{}{}, // depends on unconverted modules: truth-prebuilt
		"truth-prebuilt":                         struct{}{}, // depends on unconverted modules: asm-7.0, guava

		// b/215723302; awaiting tz{data,_version} to then rename targets conflicting with srcs
		"tzdata":     struct{}{},
		"tz_version": struct{}{},

		// '//bionic/libc:libc_bp2build_cc_library_static' is duplicated in the 'deps' attribute of rule
		"toybox-static": struct{}{},

		// Do not convert the following modules because of duplicate labels checking in Bazel.
		// See b/241283350. They should be removed from this list once the bug is fixed.
		"libartpalette":        struct{}{},
		"libartbase":           struct{}{},
		"libdexfile":           struct{}{},
		"libartbased":          struct{}{},
		"libdexfile_static":    struct{}{},
		"libartbase-testing":   struct{}{},
		"libartbased-testing":  struct{}{},
		"libdexfile_support":   struct{}{},
		"libunwindstack":       struct{}{},
		"libunwindstack_local": struct{}{},
		"libfdtrack":           struct{}{},
		"libc_malloc_debug":    struct{}{},
		"libutilscallstack":    struct{}{},
		"libunwindstack_utils": struct{}{},
		"unwind_for_offline":   struct{}{},
	}

	Bp2buildCcLibraryStaticOnlyList = map[string]struct{}{}

	MixedBuildsDisabledList = map[string]struct{}{
		// TODO(b/237315968); Depend on prebuilt stl, not from source
		"libruy_static":          struct{}{},
		"libtflite_kernel_utils": struct{}{},

		"art_libdexfile_dex_instruction_list_header": struct{}{}, // breaks libart_mterp.armng, header not found

		"libbrotli":               struct{}{}, // http://b/198585397: ld.lld: error: bionic/libc/arch-arm64/generic/bionic/memmove.S:95:(.text+0x10): relocation R_AARCH64_CONDBR19 out of range: -1404176 is not in [-1048576, 1048575]; references __memcpy
		"minijail_constants_json": struct{}{}, // http://b/200899432: bazel-built cc_genrule does not work in mixed build when it is a dependency of another soong module.

		"cap_names.h": struct{}{}, // TODO(b/204913827) runfiles need to be handled in mixed builds
		"libcap":      struct{}{}, // TODO(b/204913827) runfiles need to be handled in mixed builds
		// Unsupported product&vendor suffix. b/204811222 and b/204810610.
		"libprotobuf-cpp-full": struct{}{},
		"libprotobuf-cpp-lite": struct{}{},

		// Depends on libprotobuf-cpp-*
		"libadb_pairing_connection":        struct{}{},
		"libadb_pairing_connection_static": struct{}{},
		"libadb_pairing_server":            struct{}{},
		"libadb_pairing_server_static":     struct{}{},

		// TODO(b/204811222) support suffix in cc_binary
		"acvp_modulewrapper":                         struct{}{},
		"android.hardware.media.c2@1.0-service-v4l2": struct{}{},
		"app_process":                                struct{}{},
		"bar_test":                                   struct{}{},
		"bench_cxa_atexit":                           struct{}{},
		"bench_noop":                                 struct{}{},
		"bench_noop_nostl":                           struct{}{},
		"bench_noop_static":                          struct{}{},
		"boringssl_self_test":                        struct{}{},
		"boringssl_self_test_vendor":                 struct{}{},
		"bssl":                                       struct{}{},
		"cavp":                                       struct{}{},
		"crash_dump":                                 struct{}{},
		"crasher":                                    struct{}{},
		"libcxx_test_template":                       struct{}{},
		"linker":                                     struct{}{},
		"memory_replay":                              struct{}{},
		"native_bridge_guest_linker":                 struct{}{},
		"native_bridge_stub_library_defaults":        struct{}{},
		"noop":                                       struct{}{},
		"simpleperf_ndk":                             struct{}{},
		"toybox-static":                              struct{}{},
		"zlib_bench":                                 struct{}{},

		// java_import[_host] issues
		// tradefed prebuilts depend on libprotobuf
		"prebuilt_tradefed":                struct{}{},
		"prebuilt_tradefed-test-framework": struct{}{},
		// handcrafted BUILD.bazel files in //prebuilts/...
		"prebuilt_r8lib-prebuilt":                             struct{}{},
		"prebuilt_sdk-core-lambda-stubs":                      struct{}{},
		"prebuilt_android-support-collections-nodeps":         struct{}{},
		"prebuilt_android-arch-core-common-nodeps":            struct{}{},
		"prebuilt_android-arch-lifecycle-common-java8-nodeps": struct{}{},
		"prebuilt_android-arch-lifecycle-common-nodeps":       struct{}{},
		"prebuilt_android-support-annotations-nodeps":         struct{}{},
		"prebuilt_android-arch-paging-common-nodeps":          struct{}{},
		"prebuilt_android-arch-room-common-nodeps":            struct{}{},
		// TODO(b/217750501) exclude_dirs property not supported
		"prebuilt_kotlin-reflect":     struct{}{},
		"prebuilt_kotlin-stdlib":      struct{}{},
		"prebuilt_kotlin-stdlib-jdk7": struct{}{},
		"prebuilt_kotlin-stdlib-jdk8": struct{}{},
		"prebuilt_kotlin-test":        struct{}{},
		// TODO(b/217750501) exclude_files property not supported
		"prebuilt_platform-robolectric-4.4-prebuilt":   struct{}{},
		"prebuilt_platform-robolectric-4.5.1-prebuilt": struct{}{},
		"prebuilt_currysrc_org.eclipse":                struct{}{},
	}

	ProdMixedBuildsEnabledList = map[string]struct{}{
		// This list left intentionally empty for now. Add specific module names
		// to have them built by Bazel in Prod Mixed Builds mode.
	}
)
