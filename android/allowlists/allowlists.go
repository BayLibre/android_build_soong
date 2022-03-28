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
		"art/libdexfile":                        Bp2BuildDefaultTrueRecursively,
		"art/runtime":                           Bp2BuildDefaultTrueRecursively,
		"art/tools":                             Bp2BuildDefaultTrue,
		"bionic":                                Bp2BuildDefaultTrueRecursively,
		"bootable/recovery/tools/recovery_l10n": Bp2BuildDefaultTrue,
		"build/bazel/examples/soong_config_variables":        Bp2BuildDefaultTrueRecursively,
		"build/bazel/examples/apex/minimal":                  Bp2BuildDefaultTrueRecursively,
		"build/make/tools/signapk":                           Bp2BuildDefaultTrue,
		"build/make/target/product/security":                 Bp2BuildDefaultTrue,
		"build/soong":                                        Bp2BuildDefaultTrue,
		"build/soong/cc/libbuildversion":                     Bp2BuildDefaultTrue, // Skip tests subdir
		"build/soong/cc/ndkstubgen":                          Bp2BuildDefaultTrue,
		"build/soong/cc/symbolfile":                          Bp2BuildDefaultTrue,
		"build/soong/linkerconfig":                           Bp2BuildDefaultTrueRecursively,
		"build/soong/scripts":                                Bp2BuildDefaultTrueRecursively,
		"cts/common/device-side/nativetesthelper/jni":        Bp2BuildDefaultTrueRecursively,
		"development/apps/DevelopmentSettings":               Bp2BuildDefaultTrue,
		"development/apps/Fallback":                          Bp2BuildDefaultTrue,
		"development/apps/WidgetPreview":                     Bp2BuildDefaultTrue,
		"development/samples/BasicGLSurfaceView":             Bp2BuildDefaultTrue,
		"development/samples/BluetoothChat":                  Bp2BuildDefaultTrue,
		"development/samples/BrokenKeyDerivation":            Bp2BuildDefaultTrue,
		"development/samples/Compass":                        Bp2BuildDefaultTrue,
		"development/samples/ContactManager":                 Bp2BuildDefaultTrue,
		"development/samples/FixedGridLayout":                Bp2BuildDefaultTrue,
		"development/samples/HelloEffects":                   Bp2BuildDefaultTrue,
		"development/samples/Home":                           Bp2BuildDefaultTrue,
		"development/samples/HoneycombGallery":               Bp2BuildDefaultTrue,
		"development/samples/JetBoy":                         Bp2BuildDefaultTrue,
		"development/samples/KeyChainDemo":                   Bp2BuildDefaultTrue,
		"development/samples/LceDemo":                        Bp2BuildDefaultTrue,
		"development/samples/LunarLander":                    Bp2BuildDefaultTrue,
		"development/samples/MultiResolution":                Bp2BuildDefaultTrue,
		"development/samples/MultiWindow":                    Bp2BuildDefaultTrue,
		"development/samples/NotePad":                        Bp2BuildDefaultTrue,
		"development/samples/Obb":                            Bp2BuildDefaultTrue,
		"development/samples/RSSReader":                      Bp2BuildDefaultTrue,
		"development/samples/ReceiveShareDemo":               Bp2BuildDefaultTrue,
		"development/samples/SearchableDictionary":           Bp2BuildDefaultTrue,
		"development/samples/SipDemo":                        Bp2BuildDefaultTrue,
		"development/samples/SkeletonApp":                    Bp2BuildDefaultTrue,
		"development/samples/Snake":                          Bp2BuildDefaultTrue,
		"development/samples/SpellChecker/":                  Bp2BuildDefaultTrueRecursively,
		"development/samples/ThemedNavBarKeyboard":           Bp2BuildDefaultTrue,
		"development/samples/ToyVpn":                         Bp2BuildDefaultTrue,
		"development/samples/TtsEngine":                      Bp2BuildDefaultTrue,
		"development/samples/USB/AdbTest":                    Bp2BuildDefaultTrue,
		"development/samples/USB/MissileLauncher":            Bp2BuildDefaultTrue,
		"development/samples/VoiceRecognitionService":        Bp2BuildDefaultTrue,
		"development/samples/VoicemailProviderDemo":          Bp2BuildDefaultTrue,
		"development/samples/WiFiDirectDemo":                 Bp2BuildDefaultTrue,
		"development/sdk":                                    Bp2BuildDefaultTrueRecursively,
		"external/arm-optimized-routines":                    Bp2BuildDefaultTrueRecursively,
		"external/auto/android-annotation-stubs":             Bp2BuildDefaultTrueRecursively,
		"external/auto/common":                               Bp2BuildDefaultTrueRecursively,
		"external/auto/service":                              Bp2BuildDefaultTrueRecursively,
		"external/boringssl":                                 Bp2BuildDefaultTrueRecursively,
		"external/bouncycastle":                              Bp2BuildDefaultTrue,
		"external/brotli":                                    Bp2BuildDefaultTrue,
		"external/conscrypt":                                 Bp2BuildDefaultTrue,
		"external/e2fsprogs":                                 Bp2BuildDefaultTrueRecursively,
		"external/error_prone":                               Bp2BuildDefaultTrueRecursively,
		"external/fmtlib":                                    Bp2BuildDefaultTrueRecursively,
		"external/google-benchmark":                          Bp2BuildDefaultTrueRecursively,
		"external/googletest":                                Bp2BuildDefaultTrueRecursively,
		"external/gwp_asan":                                  Bp2BuildDefaultTrueRecursively,
		"external/icu":                                       Bp2BuildDefaultTrueRecursively,
		"external/icu/android_icu4j":                         Bp2BuildDefaultFalse, // java rules incomplete
		"external/icu/icu4j":                                 Bp2BuildDefaultFalse, // java rules incomplete
		"external/javapoet":                                  Bp2BuildDefaultTrueRecursively,
		"external/jemalloc_new":                              Bp2BuildDefaultTrueRecursively,
		"external/jsoncpp":                                   Bp2BuildDefaultTrueRecursively,
		"external/libcap":                                    Bp2BuildDefaultTrueRecursively,
		"external/libcxx":                                    Bp2BuildDefaultTrueRecursively,
		"external/libcxxabi":                                 Bp2BuildDefaultTrueRecursively,
		"external/libevent":                                  Bp2BuildDefaultTrueRecursively,
		"external/libpng":                                    Bp2BuildDefaultTrueRecursively,
		"external/lz4/lib":                                   Bp2BuildDefaultTrue,
		"external/lzma/C":                                    Bp2BuildDefaultTrueRecursively,
		"external/mdnsresponder":                             Bp2BuildDefaultTrueRecursively,
		"external/minijail":                                  Bp2BuildDefaultTrueRecursively,
		"external/pcre":                                      Bp2BuildDefaultTrueRecursively,
		"external/protobuf":                                  Bp2BuildDefaultTrueRecursively,
		"external/python/six":                                Bp2BuildDefaultTrueRecursively,
		"external/scudo":                                     Bp2BuildDefaultTrueRecursively,
		"external/selinux/libselinux":                        Bp2BuildDefaultTrueRecursively,
		"external/selinux/libsepol":                          Bp2BuildDefaultTrueRecursively,
		"external/zlib":                                      Bp2BuildDefaultTrueRecursively,
		"external/zstd":                                      Bp2BuildDefaultTrueRecursively,
		"frameworks/base/media/tests/MediaDump":              Bp2BuildDefaultTrue,
		"frameworks/base/startop/apps/test":                  Bp2BuildDefaultTrue,
		"frameworks/base/tests/appwidgets/AppWidgetHostTest": Bp2BuildDefaultTrueRecursively,
		"frameworks/native/libs/adbd_auth":                   Bp2BuildDefaultTrueRecursively,
		"frameworks/native/opengl/tests/gl2_cameraeye":       Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/gl2_java":            Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/testLatency":         Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/testPauseResume":     Bp2BuildDefaultTrue,
		"frameworks/native/opengl/tests/testViewport":        Bp2BuildDefaultTrue,
		"frameworks/proto_logging/stats/stats_log_api_gen":   Bp2BuildDefaultTrueRecursively,
		"libnativehelper":                                    Bp2BuildDefaultTrueRecursively,
		"packages/apps/DevCamera":                            Bp2BuildDefaultTrue,
		"packages/apps/HTMLViewer":                           Bp2BuildDefaultTrue,
		"packages/apps/Protips":                              Bp2BuildDefaultTrue,
		"packages/modules/StatsD/lib/libstatssocket":         Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb":                               Bp2BuildDefaultTrue,
		"packages/modules/adb/apex":                          Bp2BuildDefaultTrue,
		"packages/modules/adb/crypto":                        Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/libs":                          Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/pairing_auth":                  Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/pairing_connection":            Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/proto":                         Bp2BuildDefaultTrueRecursively,
		"packages/modules/adb/tls":                           Bp2BuildDefaultTrueRecursively,
		"packages/providers/MediaProvider/tools/dialogs":     Bp2BuildDefaultTrue,
		"packages/screensavers/Basic":                        Bp2BuildDefaultTrue,
		"packages/services/Car/tests/SampleRearViewCamera":   Bp2BuildDefaultTrue,
		"prebuilts/clang/host/linux-x86":                     Bp2BuildDefaultTrueRecursively,
		"prebuilts/tools/common/m2":                          Bp2BuildDefaultTrue,
		"system/apex":                                        Bp2BuildDefaultFalse, // TODO(b/207466993): flaky failures
		"system/apex/proto":                                  Bp2BuildDefaultTrueRecursively,
		"system/apex/libs":                                   Bp2BuildDefaultTrueRecursively,
		"system/core/debuggerd":                              Bp2BuildDefaultTrueRecursively,
		"system/core/diagnose_usb":                           Bp2BuildDefaultTrueRecursively,
		"system/core/libasyncio":                             Bp2BuildDefaultTrue,
		"system/core/libcrypto_utils":                        Bp2BuildDefaultTrueRecursively,
		"system/core/libcutils":                              Bp2BuildDefaultTrueRecursively,
		"system/core/libpackagelistparser":                   Bp2BuildDefaultTrueRecursively,
		"system/core/libprocessgroup":                        Bp2BuildDefaultTrue,
		"system/core/libprocessgroup/cgrouprc":               Bp2BuildDefaultTrue,
		"system/core/libprocessgroup/cgrouprc_format":        Bp2BuildDefaultTrue,
		"system/core/libsystem":                              Bp2BuildDefaultTrueRecursively,
		"system/core/libutils":                               Bp2BuildDefaultTrueRecursively,
		"system/core/libvndksupport":                         Bp2BuildDefaultTrueRecursively,
		"system/core/property_service/libpropertyinfoparser": Bp2BuildDefaultTrueRecursively,
		"system/libbase":                                     Bp2BuildDefaultTrueRecursively,
		"system/libprocinfo":                                 Bp2BuildDefaultTrue,
		"system/libziparchive":                               Bp2BuildDefaultTrueRecursively,
		"system/logging/liblog":                              Bp2BuildDefaultTrueRecursively,
		"system/sepolicy/apex":                               Bp2BuildDefaultTrueRecursively,
		"system/timezone/apex":                               Bp2BuildDefaultTrueRecursively,
		"system/timezone/output_data":                        Bp2BuildDefaultTrueRecursively,
		"system/unwinding/libbacktrace":                      Bp2BuildDefaultTrueRecursively,
		"system/unwinding/libunwindstack":                    Bp2BuildDefaultTrueRecursively,
		"tools/apksig":                                       Bp2BuildDefaultTrue,
		"tools/platform-compat/java/android/compat":          Bp2BuildDefaultTrueRecursively,
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
		"build/bazel/examples/java":/* recursive = */ true,
		"build/bazel/bazel_skylib":/* recursive = */ true,
		"build/bazel/rules":/* recursive = */ true,
		"build/bazel/rules_cc":/* recursive = */ true,
		"build/bazel/scripts":/* recursive = */ true,
		"build/bazel/tests":/* recursive = */ true,
		"build/bazel/platforms":/* recursive = */ true,
		"build/bazel/product_variables":/* recursive = */ true,
		"build/bazel/vendor/google":/* recursive = */ true,
		"build/bazel_common_rules":/* recursive = */ true,
		// build/make/tools/signapk BUILD file is generated, so build/make/tools is not recursive.
		"build/make/tools":/* recursive = */ false,
		"build/pesto":/* recursive = */ true,

		// external/bazelbuild-rules_android/... is needed by mixed builds, otherwise mixed builds analysis fails
		// e.g. ERROR: Analysis of target '@soong_injection//mixed_builds:buildroot' failed
		"external/bazelbuild-rules_android":/* recursive = */ true,
		"external/bazel-skylib":/* recursive = */ true,
		"external/guava":/* recursive = */ true,
		"external/jsr305":/* recursive = */ true,
		"frameworks/ex/common":/* recursive = */ true,

		"packages/apps/Music":/* recursive = */ true,
		"packages/apps/QuickSearchBox":/* recursive = */ true,
		"packages/apps/WallpaperPicker":/* recursive = */ false,

		"prebuilts/bundletool":/* recursive = */ true,
		"prebuilts/gcc":/* recursive = */ true,
		"prebuilts/build-tools":/* recursive = */ false,
		"prebuilts/jdk/jdk11":/* recursive = */ false,
		"prebuilts/sdk":/* recursive = */ false,
		"prebuilts/sdk/current/extras/app-toolkit":/* recursive = */ false,
		"prebuilts/sdk/current/support":/* recursive = */ false,
		"prebuilts/sdk/tools":/* recursive = */ false,
		"prebuilts/r8":/* recursive = */ false,
	}

	Bp2buildModuleAlwaysConvertList = []string{
		//external/avb
		"avbtool",
		"libavb",
		"avb_headers",

		//external/fec
		"libfec_rs",

		//system/core/libsparse
		"libsparse",

		//system/extras/ext4_utils
		"libext4_utils",

		//system/extras/libfec
		"libfec",

		//system/extras/squashfs_utils
		"libsquashfs_utils",

		//system/extras/verity/fec
		"fec",

		//packages/apps/Car/libs/car-ui-lib/car-ui-androidx
		// genrule dependencies for java_imports
		"car-ui-androidx-annotation-nodeps",
		"car-ui-androidx-collection-nodeps",
		"car-ui-androidx-core-common-nodeps",
		"car-ui-androidx-lifecycle-common-nodeps",
		"car-ui-androidx-constraintlayout-solver-nodeps",
	}

	Bp2buildModuleTypeAlwaysConvertList = []string{
		"java_import",
		"java_import_host",
	}

	Bp2buildModuleDoNotConvertList = []string{
		"libnativehelper_compat_libc", // Broken compile: implicit declaration of function 'strerror_r' is invalid in C99

		"libart",                             // depends on unconverted modules: art_operator_srcs, libodrstatslog, libelffile, art_cmdlineparser_headers, cpp-define-generator-definitions, libcpu_features, libdexfile, libartpalette, libbacktrace, libnativebridge, libnativeloader, libsigchain, libunwindstack, libartbase, libprofile, cpp-define-generator-asm-support, apex-info-list-tinyxml, libtinyxml2, libnativeloader-headers, libstatssocket, heapprofd_client_api
		"libart-runtime-gtest",               // depends on unconverted modules: libgtest_isolated, libart-compiler, libdexfile, libprofile, libartbase, libbacktrace, libartbase-art-gtest
		"libart_headers",                     // depends on unconverted modules: art_libartbase_headers
		"libartd",                            // depends on unconverted modules: apex-info-list-tinyxml, libtinyxml2, libnativeloader-headers, libstatssocket, heapprofd_client_api, art_operator_srcs, libodrstatslog, libelffiled, art_cmdlineparser_headers, cpp-define-generator-definitions, libcpu_features, libdexfiled, libartpalette, libbacktrace, libnativebridge, libnativeloader, libsigchain, libunwindstack, libartbased, libprofiled, cpp-define-generator-asm-support
		"libartd-runtime-gtest",              // depends on unconverted modules: libgtest_isolated, libartd-compiler, libdexfiled, libprofiled, libartbased, libbacktrace, libartbased-art-gtest
		"libstatslog_art",                    // depends on unconverted modules: statslog_art.cpp, statslog_art.h
		"statslog_art.h", "statslog_art.cpp", // depends on unconverted modules: stats-log-api-gen

		"libandroid_runtime_lazy", // depends on unconverted modules: libbinder_headers
		"libcmd",                  // depends on unconverted modules: libbinder

		"libdexfile_support_static",                                                       // Depends on unconverted module: libdexfile_external_headers
		"libunwindstack_local", "libunwindstack_utils", "libc_malloc_debug", "libfdtrack", // Depends on unconverted module: libunwindstack

		"libdexfile_support",          // TODO(b/210546943): Enabled based on product variables.
		"libdexfile_external_headers", // TODO(b/210546943): Enabled based on product variables.

		"libunwindstack",                // Depends on unconverted module libdexfile_support.
		"libnativehelper_compat_libc++", // Broken compile: implicit declaration of function 'strerror_r' is invalid in C99

		"chkcon", "sefcontext_compile", // depends on unconverted modules: libsepol

		"libsepol", // TODO(b/207408632): Unsupported case of .l sources in cc library rules

		"gen-kotlin-build-file.py", // module has same name as source

		"libactivitymanager_aidl", // TODO(b/207426160): Depends on activity_manager_procstate_aidl, which is an aidl filegroup.

		"libnativehelper_lazy_mts_jni", "libnativehelper_mts_jni", // depends on unconverted modules: libgmock_ndk
		"libnativetesthelper_jni", "libgmock_main_ndk", "libgmock_ndk", // depends on unconverted module: libgtest_ndk_c++

		"statslog-framework-java-gen", "statslog.cpp", "statslog.h", "statslog.rs", "statslog_header.rs", // depends on unconverted modules: stats-log-api-gen

		"stats-log-api-gen", // depends on unconverted modules: libstats_proto_host, libprotobuf-cpp-full

		"libstatslog", // depends on unconverted modules: statslog.cpp, statslog.h, ...

		"cmd",                                                        // depends on unconverted module packagemanager_aidl-cpp, of unsupported type aidl_interface
		"servicedispatcher",                                          // depends on unconverted module android.debug_aidl, of unsupported type aidl_interface
		"libutilscallstack",                                          // depends on unconverted module libbacktrace
		"libbacktrace",                                               // depends on unconverted module libunwindstack
		"libdebuggerd_handler",                                       // depends on unconverted module libdebuggerd_handler_core
		"libdebuggerd_handler_core", "libdebuggerd_handler_fallback", // depends on unconverted module libdebuggerd
		"unwind_for_offline", // depends on unconverted module libunwindstack_utils
		"libdebuggerd",       // depends on unconverted modules libdexfile_support, libunwindstack, gwp_asan_crash_handler, libtombstone_proto, libprotobuf-cpp-lite
		"libdexfile_static",  // depends on libartpalette, libartbase, libdexfile, which are of unsupported type: art_cc_library.

		"static_crasher", // depends on unconverted modules: libdebuggerd_handler

		"pbtombstone", "crash_dump", // depends on libdebuggerd, libunwindstack

		"libbase_ndk", // http://b/186826477, fails to link libctscamera2_jni for device (required for CtsCameraTestCases)

		"libprotobuf-internal-protos",      // b/210751803, we don't handle path property for filegroups
		"libprotobuf-internal-python-srcs", // b/210751803, we don't handle path property for filegroups
		"libprotobuf-java-full",            // b/210751803, we don't handle path property for filegroups
		"host-libprotobuf-java-full",       // b/210751803, we don't handle path property for filegroups
		"libprotobuf-java-util-full",       // b/210751803, we don't handle path property for filegroups
		"apex_manifest_proto_java",         // b/210751803, depends on libprotobuf-java-full
		"conscrypt",                        // b/210751803, we don't handle path property for filegroups
		"conscrypt-for-host",               // b/210751803, we don't handle path property for filegroups

		"libprotobuf-java-nano",      // b/220869005, depends on non-public_current SDK
		"host-libprotobuf-java-nano", // b/220869005, depends on libprotobuf-java-nano

		"libc_musl_sysroot_bionic_arch_headers", // b/218405924, depends on soong_zip
		"libc_musl_sysroot_bionic_headers",      // b/218405924, depends on soong_zip and generates duplicate srcs

		// python protos
		"libprotobuf-python",                           // contains .proto sources
		"conv_linker_config",                           // depends on linker_config_proto, a python lib with proto sources
		"apex_build_info_proto", "apex_manifest_proto", // a python lib with proto sources
		"linker_config_proto", // contains .proto sources

		"brotli-fuzzer-corpus", // b/202015218: outputs are in location incompatible with bazel genrule handling.

		// python modules
		"analyze_bcpf", // depends on bpmodify a blueprint_go_binary.

		// b/203369847: multiple genrules in the same package creating the same file
		// //development/sdk/...
		"platform_tools_properties",
		"build_tools_source_properties",

		// APEX support
		"com.android.runtime", // depends on unconverted modules: bionic-linker-config, linkerconfig

		"libgtest_ndk_c++",      // b/201816222: Requires sdk_version support.
		"libgtest_main_ndk_c++", // b/201816222: Requires sdk_version support.

		"abb",                     // depends on unconverted modules: libcmd, libbinder
		"adb",                     // depends on unconverted modules: AdbWinApi, libadb_host, libandroidfw, libapp_processes_protos_full, libfastdeploy_host, libopenscreen-discovery, libopenscreen-platform-impl, libusb, bin2c_fastdeployagent, AdbWinUsbApi
		"libadb_host",             // depends on unconverted modules: libopenscreen-discovery, libopenscreen-platform-impl, libusb, AdbWinApi
		"libfastdeploy_host",      // depends on unconverted modules: libandroidfw, libusb, AdbWinApi
		"linker",                  // depends on unconverted modules: libdebuggerd_handler_fallback
		"linker_reloc_bench_main", // depends on unconverted modules: liblinker_reloc_bench_*
		"versioner",               // depends on unconverted modules: libclang_cxx_host, libLLVM_host, of unsupported type llvm_host_prebuilt_library_shared

		"linkerconfig", // http://b/202876379 has arch-variant static_executable
		"mdnsd",        // http://b/202876379 has arch-variant static_executable

		"CarHTMLViewer", // depends on unconverted modules android.car-stubs, car-ui-lib

		"libdexfile",  // depends on unconverted modules: dexfile_operator_srcs, libartbase, libartpalette,
		"libdexfiled", // depends on unconverted modules: dexfile_operator_srcs, libartbased, libartpalette

		// go deps:
		"apex-protos",                                                                                // depends on soong_zip, a go binary
		"generated_android_icu4j_src_files", "generated_android_icu4j_test_files", "icu4c_test_data", // depends on unconverted modules: soong_zip
		"host_bionic_linker_asm",                                                  // depends on extract_linker, a go binary.
		"host_bionic_linker_script",                                               // depends on extract_linker, a go binary.
		"robolectric-sqlite4java-native",                                          // depends on soong_zip, a go binary
		"robolectric_tzdata",                                                      // depends on soong_zip, a go binary
		"libc_musl_sysroot_libc++_headers", "libc_musl_sysroot_libc++abi_headers", // depends on soong_zip, zip2zip

		"android_icu4j_srcgen_binary", // Bazel build error: deps not allowed without srcs; move to runtime_deps
		"core-icu4j-for-host",         // Bazel build error: deps not allowed without srcs; move to runtime_deps

		// java deps
		"android_icu4j_srcgen",          // depends on unconverted modules: currysrc
		"bin2c_fastdeployagent",         // depends on deployagent, a java binary
		"currysrc",                      // depends on unconverted modules: currysrc_org.eclipse, guavalib, jopt-simple-4.9
		"robolectric-sqlite4java-0.282", // depends on unconverted modules: robolectric-sqlite4java-import, robolectric-sqlite4java-native
		"timezone-host",                 // depends on unconverted modules: art.module.api.annotations
		"truth-host-prebuilt",           // depends on unconverted modules: truth-prebuilt
		"truth-prebuilt",                // depends on unconverted modules: asm-7.0, guava

		"generated_android_icu4j_resources",      // depends on unconverted modules: android_icu4j_srcgen_binary, soong_zip
		"generated_android_icu4j_test_resources", // depends on unconverted modules: android_icu4j_srcgen_binary, soong_zip

		"art-script",     // depends on unconverted modules: dalvikvm, dex2oat
		"dex2oat-script", // depends on unconverted modules: dex2oat

		"prebuilt_car-ui-androidx-core-common",         // b/224773339, genrule dependency creates an .aar, not a .jar
		"prebuilt_platform-robolectric-4.4-prebuilt",   // aosp/1999250, needs .aar support in Jars
		"prebuilt_platform-robolectric-4.5.1-prebuilt", // aosp/1999250, needs .aar support in Jars
	}

	Bp2buildCcLibraryStaticOnlyList = []string{}

	MixedBuildsDisabledList = []string{
		"art_libdexfile_dex_instruction_list_header", // breaks libart_mterp.armng, header not found

		"libbrotli",               // http://b/198585397, ld.lld: error: bionic/libc/arch-arm64/generic/bionic/memmove.S:95:(.text+0x10): relocation R_AARCH64_CONDBR19 out of range: -1404176 is not in [-1048576, 1048575]; references __memcpy
		"minijail_constants_json", // http://b/200899432, bazel-built cc_genrule does not work in mixed build when it is a dependency of another soong module.

		"cap_names.h",                                  // TODO(b/204913827) runfiles need to be handled in mixed builds
		"libcap",                                       // TODO(b/204913827) runfiles need to be handled in mixed builds
		"libprotobuf-cpp-full", "libprotobuf-cpp-lite", // Unsupported product&vendor suffix. b/204811222 and b/204810610.

		// Depends on libprotobuf-cpp-*
		"libadb_pairing_connection",
		"libadb_pairing_connection_static",
		"libadb_pairing_server", "libadb_pairing_server_static",

		// TODO(b/204811222) support suffix in cc_binary
		"acvp_modulewrapper",
		"android.hardware.media.c2@1.0-service-v4l2",
		"app_process",
		"bar_test",
		"bench_cxa_atexit",
		"bench_noop",
		"bench_noop_nostl",
		"bench_noop_static",
		"boringssl_self_test",
		"boringssl_self_test_vendor",
		"bssl",
		"cavp",
		"crash_dump",
		"crasher",
		"libcxx_test_template",
		"linker",
		"memory_replay",
		"native_bridge_guest_linker",
		"native_bridge_stub_library_defaults",
		"noop",
		"simpleperf_ndk",
		"toybox-static",
		"zlib_bench",
	}
)
