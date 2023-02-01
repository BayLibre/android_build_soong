// Copyright 2021 The Android Open Source Project
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

package java

import (
	"path/filepath"
	"runtime"
	"testing"

	"android/soong/android"
	"android/soong/cc"
)

var prepForJavaFuzzTest = android.GroupFixturePreparers(
	PrepareForTestWithJavaDefaultModules,
	cc.PrepareForTestWithCcBuildComponents,
	android.FixtureRegisterWithContext(RegisterJavaFuzzBuildComponents),
)

func TestJavaFuzz(t *testing.T) {
	result := prepForJavaFuzzTest.RunTestWithBp(t, `
		java_fuzz {
			name: "foo",
			host_supported: true,
			device_supported: false,
			srcs: ["a.java"],
			libs: ["bar"],
			static_libs: ["baz"],
            jni_libs: [
                "libjni",
            ],
		}

		java_library_host {
			name: "bar",
			srcs: ["b.java"],
		}

		java_library_host {
			name: "baz",
			srcs: ["c.java"],
		}

		cc_library_shared {
			name: "libjni",
			host_supported: true,
			device_supported: false,
			stl: "none",
		}
		`)

	osCommonTarget := result.Config.BuildOSCommonTarget.String()

	javac := result.ModuleForTests("foo", osCommonTarget).Rule("javac")
	combineJar := result.ModuleForTests("foo", osCommonTarget).Description("for javac")

	if len(javac.Inputs) != 1 || javac.Inputs[0].String() != "a.java" {
		t.Errorf(`foo inputs %v != ["a.java"]`, javac.Inputs)
	}

	baz := result.ModuleForTests("baz", osCommonTarget).Rule("javac").Output.String()
	barOut := filepath.Join("out", "soong", ".intermediates", "bar", osCommonTarget, "javac", "bar.jar")
	bazOut := filepath.Join("out", "soong", ".intermediates", "baz", osCommonTarget, "javac", "baz.jar")

	android.AssertStringDoesContain(t, "foo classpath", javac.Args["classpath"], barOut)
	android.AssertStringDoesContain(t, "foo classpath", javac.Args["classpath"], bazOut)

	if len(combineJar.Inputs) != 2 || combineJar.Inputs[1].String() != baz {
		t.Errorf("foo combineJar inputs %v does not contain %q", combineJar.Inputs, baz)
	}

	ctx := result.TestContext
	foo := ctx.ModuleForTests("foo", osCommonTarget).Module().(*JavaFuzzTest)

	expected := "lib64/libjni.so"
	if runtime.GOOS == "darwin" {
		expected = "lib64/libjni.dylib"
	}

	fooJniFilePaths := foo.jniFilePaths
	if len(fooJniFilePaths) != 1 || fooJniFilePaths[0].Rel() != expected {
		t.Errorf(`expected foo test data relative path [%q], got %q`,
			expected, fooJniFilePaths.Strings())
	}
}

func TestJavaFuzz_FULL(t *testing.T) {
	result := prepForJavaFuzzTest.RunTestWithBp(t, `
		java_fuzz {
			name: "foo",
			host_supported: true,
			device_supported: true,
			srcs: ["a.java"],
			jni_libs: [
				"libjni",
			],
		}

		cc_library_shared {
			name: "libjni",
			host_supported: true,
			device_supported: true,
			srcs: [
				"test.cc",
			],
			shared_libs: [
				"libjni_two",
			],
			stl: "none",
		}

		cc_library_shared {
			name: "libjni_two",
			host_supported: true,
			device_supported: true,
			srcs: [
				"test_two.cc",
			],
			stl: "none",
		}
		
		cc_defaults {
			name: "toolchain_libs_defaults",
			vendor_available: true,
			product_available: true,
			recovery_available: true,
			no_libcrt: true,
			sdk_version: "minimum",
			nocrt: true,
			system_shared_libs: [],
			stl: "none",
			check_elf_files: false,
			sanitize: {
				never: true,
			},
		}

		cc_prebuilt_library_static {
			name: "libcompiler_rt-extras",
			defaults: ["toolchain_libs_defaults"],
			vendor_ramdisk_available: true,
			srcs: [""],
		}

		cc_prebuilt_library_static {
			name: "libclang_rt.builtins",
			defaults: ["toolchain_libs_defaults"],
			host_supported: true,
	        vendor_available: true,
			vendor_ramdisk_available: true,
			native_bridge_supported: true,
			srcs: [""],
		}

		cc_prebuilt_library_shared {
			name: "libclang_rt.hwasan",
			defaults: ["toolchain_libs_defaults"],
			srcs: [""],
		}

		cc_prebuilt_library_static {
			name: "libunwind",
			defaults: [
				"linux_bionic_supported",
				"toolchain_libs_defaults",
			],
			vendor_ramdisk_available: true,
			native_bridge_supported: true,
			srcs: [""],
		}

		cc_prebuilt_library_static {
			name: "libclang_rt.fuzzer",
			defaults: [
				"linux_bionic_supported",
				"toolchain_libs_defaults",
			],
			srcs: [""],
		}

		// Needed for sanitizer
		cc_prebuilt_library_shared {
			name: "libclang_rt.ubsan_standalone",
			defaults: ["toolchain_libs_defaults"],
			srcs: [""],
		}

		cc_prebuilt_library_static {
			name: "libclang_rt.ubsan_minimal",
			defaults: ["toolchain_libs_defaults"],
			host_supported: true,
			target: {
				android_arm64: {
					srcs: ["libclang_rt.ubsan_minimal.android_arm64.a"],
				},
				android_arm: {
					srcs: ["libclang_rt.ubsan_minimal.android_arm.a"],
				},
				linux_glibc_x86_64: {
					srcs: ["libclang_rt.ubsan_minimal.x86_64.a"],
				},
				linux_glibc_x86: {
					srcs: ["libclang_rt.ubsan_minimal.x86.a"],
				},
			},
		}

		cc_library {
			name: "libc",
			defaults: ["linux_bionic_supported"],
			no_libcrt: true,
			nocrt: true,
			stl: "none",
			system_shared_libs: [],
			recovery_available: true,
			stubs: {
				versions: ["27", "28", "29"],
			},
			llndk: {
				symbol_file: "libc.map.txt",
			},
		}
		cc_library {
			name: "libm",
			defaults: ["linux_bionic_supported"],
			no_libcrt: true,
			nocrt: true,
			stl: "none",
			system_shared_libs: [],
			recovery_available: true,
			stubs: {
				versions: ["27", "28", "29"],
			},
			apex_available: [
				"//apex_available:platform",
				"myapex"
			],
			llndk: {
				symbol_file: "libm.map.txt",
			},
		}

		// Coverage libraries
		cc_library {
			name: "libprofile-extras",
			vendor_available: true,
			vendor_ramdisk_available: true,
			product_available: true,
			recovery_available: true,
			native_coverage: false,
			system_shared_libs: [],
			stl: "none",
		}
		cc_library {
			name: "libprofile-clang-extras",
			vendor_available: true,
			vendor_ramdisk_available: true,
			product_available: true,
			recovery_available: true,
			native_coverage: false,
			system_shared_libs: [],
			stl: "none",
		}
		cc_library {
			name: "libprofile-extras_ndk",
			vendor_available: true,
			product_available: true,
			native_coverage: false,
			system_shared_libs: [],
			stl: "none",
			sdk_version: "current",
		}
		cc_library {
			name: "libprofile-clang-extras_ndk",
			vendor_available: true,
			product_available: true,
			native_coverage: false,
			system_shared_libs: [],
			stl: "none",
			sdk_version: "current",
		}

		cc_library {
			name: "libdl",
			defaults: ["linux_bionic_supported"],
			no_libcrt: true,
			nocrt: true,
			stl: "none",
			system_shared_libs: [],
			recovery_available: true,
			stubs: {
				versions: ["27", "28", "29"],
			},
			apex_available: [
				"//apex_available:platform",
				"myapex"
			],
			llndk: {
				symbol_file: "libdl.map.txt",
			},
		}
		cc_library {
			name: "libft2",
			no_libcrt: true,
			nocrt: true,
			system_shared_libs: [],
			recovery_available: true,
			llndk: {
				symbol_file: "libft2.map.txt",
				private: true,
			}
		}
		cc_library {
			name: "libc++_static",
			no_libcrt: true,
			nocrt: true,
			system_shared_libs: [],
			stl: "none",
			vendor_available: true,
			vendor_ramdisk_available: true,
			product_available: true,
			recovery_available: true,
			host_supported: true,
			min_sdk_version: "29",
			apex_available: [
				"//apex_available:platform",
				"//apex_available:anyapex",
			],
		}
		cc_library {
			name: "libc++",
			no_libcrt: true,
			nocrt: true,
			system_shared_libs: [],
			stl: "none",
			vendor_available: true,
			product_available: true,
			recovery_available: true,
			host_supported: true,
			min_sdk_version: "29",
			vndk: {
				enabled: true,
				support_system_process: true,
			},
			apex_available: [
				"//apex_available:platform",
				"//apex_available:anyapex",
			],
		}
		cc_library {
			name: "libc++demangle",
			no_libcrt: true,
			nocrt: true,
			system_shared_libs: [],
			stl: "none",
			host_supported: false,
			vendor_available: true,
			vendor_ramdisk_available: true,
			product_available: true,
			recovery_available: true,
			min_sdk_version: "29",
			apex_available: [
				"//apex_available:platform",
				"//apex_available:anyapex",
			],
		}

		cc_defaults {
			name: "crt_defaults",
			defaults: ["linux_bionic_supported"],
			recovery_available: true,
			vendor_available: true,
			vendor_ramdisk_available: true,
			product_available: true,
			native_bridge_supported: true,
			stl: "none",
			min_sdk_version: "16",
			crt: true,
			system_shared_libs: [],
			apex_available: [
				"//apex_available:platform",
				"//apex_available:anyapex",
			],
		}

		cc_object {
			name: "crtbegin_so",
			defaults: ["crt_defaults"],
			srcs: ["crtbegin_so.c"],
			objs: ["crtbrand"],
		}

		cc_object {
			name: "crtbegin_dynamic",
			defaults: ["crt_defaults"],
			srcs: ["crtbegin.c"],
			objs: ["crtbrand"],
		}

		cc_object {
			name: "crtbegin_static",
			defaults: ["crt_defaults"],
			srcs: ["crtbegin.c"],
			objs: ["crtbrand"],
		}

		cc_object {
			name: "crtend_so",
			defaults: ["crt_defaults"],
			srcs: ["crtend_so.c"],
			objs: ["crtbrand"],
		}

		cc_object {
			name: "crtend_android",
			defaults: ["crt_defaults"],
			srcs: ["crtend.c"],
			objs: ["crtbrand"],
		}

		cc_object {
			name: "crtbrand",
			defaults: ["crt_defaults"],
			srcs: ["crtbrand.c"],
		}

		cc_library {
			name: "libprotobuf-cpp-lite",
		}

		cc_library {
			name: "ndk_libunwind",
			sdk_version: "minimum",
			stl: "none",
			system_shared_libs: [],
		}

		ndk_library {
			name: "libc",
			first_version: "minimum",
			symbol_file: "libc.map.txt",
		}

		ndk_library {
			name: "libm",
			first_version: "minimum",
			symbol_file: "libm.map.txt",
		}

		ndk_library {
			name: "libdl",
			first_version: "minimum",
			symbol_file: "libdl.map.txt",
		}

		ndk_prebuilt_shared_stl {
			name: "ndk_libc++_shared",
			export_include_dirs: ["ndk_libc++_shared"],
		}

		ndk_prebuilt_static_stl {
			name: "ndk_libandroid_support",
			export_include_dirs: ["ndk_libandroid_support"],
		}

		cc_library_static {
			name: "libgoogle-benchmark",
			sdk_version: "current",
			stl: "none",
			system_shared_libs: [],
		}

		cc_library_static {
			name: "note_memtag_heap_async",
		}

		cc_library_static {
			name: "note_memtag_heap_sync",
		}

		cc_library {
			name: "libc_musl",
			host_supported: true,
			no_libcrt: true,
			nocrt: true,
			system_shared_libs: [],
			stl: "none",
		}
		cc_binary {
			name: "linker",
			defaults: ["linux_bionic_supported"],
			recovery_available: true,
			stl: "none",
			nocrt: true,
			static_executable: true,
			native_coverage: false,
			system_shared_libs: [],
		}

		cc_genrule {
			name: "host_bionic_linker_script",
			host_supported: true,
			device_supported: false,
			target: {
				host: {
					enabled: false,
				},
				linux_bionic: {
					enabled: true,
				},
			},
			out: ["linker.script"],
		}

		cc_defaults {
			name: "linux_bionic_supported",
			host_supported: true,
			target: {
				host: {
					enabled: false,
				},
				linux_bionic: {
					enabled: true,
				},
			},
		}
		`)

	result.Config.BuildOSCommonTarget.String()
}
