// Copyright 2020 The Android Open Source Project
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

package rust

import (
	"android/soong/android"
	"android/soong/cmd/sbox/sbox_proto"
	"fmt"
	"sort"
	"testing"

	"github.com/google/blueprint/proptools"
	"google.golang.org/protobuf/encoding/prototext"
)

func TestSourceProviderCollision(t *testing.T) {
	testRustError(t, "multiple source providers generate the same filename output: bindings.rs", `
		rust_binary {
			name: "source_collider",
			srcs: [
				"foo.rs",
				":libbindings1",
				":libbindings2",
			],
		}
		rust_bindgen {
			name: "libbindings1",
			source_stem: "bindings",
			crate_name: "bindings1",
			wrapper_src: "src/any.h",
		}
		rust_bindgen {
			name: "libbindings2",
			source_stem: "bindings",
			crate_name: "bindings2",
			wrapper_src: "src/any.h",
		}
	`)
}

func TestCompilationOutputFiles(t *testing.T) {
	ctx := testRust(t, `
		rust_library {
			name: "libfizz_buzz",
			crate_name:"fizz_buzz",
			srcs: ["lib.rs"],
		}
		rust_binary {
			name: "fizz_buzz",
			crate_name:"fizz_buzz",
			srcs: ["lib.rs"],
		}
		rust_ffi {
			name: "librust_ffi",
			crate_name: "rust_ffi",
			srcs: ["lib.rs"],
		}
	`)
	testcases := []struct {
		testName      string
		moduleName    string
		variant       string
		expectedFiles []string
	}{
		{
			testName:   "dylib",
			moduleName: "libfizz_buzz",
			variant:    "android_arm64_armv8-a_dylib",
			expectedFiles: []string{
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so.clippy",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so.clippy.d.raw",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so.rsp",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so.rsp.d.raw",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so.rsp.a",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so.rsp.whole.a",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/unstripped/libfizz_buzz.dylib.so",
				"out/soong/target/product/test_device/system/lib64/libfizz_buzz.dylib.so",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/meta_lic",
			},
		},
		{
			testName:   "rlib dylib-std",
			moduleName: "libfizz_buzz",
			variant:    "android_arm64_armv8-a_rlib_dylib-std",
			expectedFiles: []string{
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_dylib-std/libfizz_buzz.rlib",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_dylib-std/libfizz_buzz.rlib.d.raw",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_dylib-std/libfizz_buzz.rlib.clippy",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_dylib-std/libfizz_buzz.rlib.clippy.d.raw",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_dylib-std/meta_lic",
			},
		},
		{
			testName:   "rlib rlib-std",
			moduleName: "libfizz_buzz",
			variant:    "android_arm64_armv8-a_rlib_rlib-std",
			expectedFiles: []string{
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/libfizz_buzz.rlib",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/libfizz_buzz.rlib.d.raw",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/libfizz_buzz.rlib.clippy",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/libfizz_buzz.rlib.clippy.d.raw",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/meta_lic",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/rustdoc.timestamp",
			},
		},
		{
			testName:   "rust_binary",
			moduleName: "fizz_buzz",
			variant:    "android_arm64_armv8-a",
			expectedFiles: []string{
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz.clippy",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz.clippy.d.raw",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz.rsp",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz.rsp.a",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz.rsp.d.raw",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz.rsp.whole.a",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/unstripped/fizz_buzz",
				"out/soong/target/product/test_device/system/bin/fizz_buzz",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/meta_lic",
			},
		},
		{
			testName:   "rust_ffi static",
			moduleName: "librust_ffi",
			variant:    "android_arm64_armv8-a_static",
			expectedFiles: []string{
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_static/librust_ffi.a",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_static/librust_ffi.a.d.raw",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_static/librust_ffi.a.clippy",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_static/librust_ffi.a.clippy.d.raw",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_static/meta_lic",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_static/rustdoc.timestamp",
			},
		},
		{
			testName:   "rust_ffi shared",
			moduleName: "librust_ffi",
			variant:    "android_arm64_armv8-a_shared",
			expectedFiles: []string{
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so.clippy",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so.clippy.d.raw",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so.rsp",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so.rsp.a",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so.rsp.d.raw",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so.rsp.whole.a",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/unstripped/librust_ffi.so",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/unstripped/librust_ffi.so.toc",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/meta_lic",
				"out/soong/target/product/test_device/system/lib64/librust_ffi.so",
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			modOutputs := ctx.ModuleForTests(tc.moduleName, tc.variant).AllOutputs()
			sort.Strings(tc.expectedFiles)
			sort.Strings(modOutputs)
			android.AssertStringPathsRelativeToTopEquals(
				t,
				"incorrect outputs from rust module",
				ctx.Config(),
				tc.expectedFiles,
				modOutputs,
			)
		})
	}
}

var (
	sboxCompilationFiles = []string{
		"out/soong/.intermediates/defaults/rust/libaddr2line/android_arm64_armv8-a_rlib/libaddr2line.rlib",
		"out/soong/.intermediates/defaults/rust/libadler/android_arm64_armv8-a_rlib/libadler.rlib",
		"out/soong/.intermediates/defaults/rust/liballoc/android_arm64_armv8-a_rlib/liballoc.rlib",
		"out/soong/.intermediates/defaults/rust/libcfg_if/android_arm64_armv8-a_rlib/libcfg_if.rlib",
		"out/soong/.intermediates/defaults/rust/libcompiler_builtins/android_arm64_armv8-a_rlib/libcompiler_builtins.rlib",
		"out/soong/.intermediates/defaults/rust/libcore/android_arm64_armv8-a_rlib/libcore.rlib",
		"out/soong/.intermediates/defaults/rust/libgimli/android_arm64_armv8-a_rlib/libgimli.rlib",
		"out/soong/.intermediates/defaults/rust/libhashbrown/android_arm64_armv8-a_rlib/libhashbrown.rlib",
		"out/soong/.intermediates/defaults/rust/liblibc/android_arm64_armv8-a_rlib/liblibc.rlib",
		"out/soong/.intermediates/defaults/rust/libmemchr/android_arm64_armv8-a_rlib/libmemchr.rlib",
		"out/soong/.intermediates/defaults/rust/libminiz_oxide/android_arm64_armv8-a_rlib/libminiz_oxide.rlib",
		"out/soong/.intermediates/defaults/rust/libobject/android_arm64_armv8-a_rlib/libobject.rlib",
		"out/soong/.intermediates/defaults/rust/libpanic_unwind/android_arm64_armv8-a_rlib/libpanic_unwind.rlib",
		"out/soong/.intermediates/defaults/rust/librustc_demangle/android_arm64_armv8-a_rlib/librustc_demangle.rlib",
		"out/soong/.intermediates/defaults/rust/librustc_std_workspace_alloc/android_arm64_armv8-a_rlib/librustc_std_workspace_alloc.rlib",
		"out/soong/.intermediates/defaults/rust/librustc_std_workspace_core/android_arm64_armv8-a_rlib/librustc_std_workspace_core.rlib",
		"out/soong/.intermediates/defaults/rust/libstd_detect/android_arm64_armv8-a_rlib/libstd_detect.rlib",
		"build/soong/scripts/mkcratersp.py",
		"defaults/rust/linux-x86/1.69.0/bin/rustc",
		"defaults/rust/linux-x86/1.69.0/lib/libstd.so",
		"defaults/rust/linux-x86/1.69.0/lib64/libc++.so.1",
	}
	sboxCompilationFilesWithCc = []string{
		"defaults/cc/common",
		"out/soong/.intermediates/defaults/cc/common/libc/android_arm64_armv8-a_shared/libc.so",
		"out/soong/.intermediates/defaults/cc/common/libc/android_arm64_armv8-a_shared/libc.so.toc",
		"out/soong/.intermediates/defaults/cc/common/libdl/android_arm64_armv8-a_shared/libdl.so",
		"out/soong/.intermediates/defaults/cc/common/libdl/android_arm64_armv8-a_shared/libdl.so.toc",
		"out/soong/.intermediates/defaults/cc/common/libm/android_arm64_armv8-a_shared/libm.so",
		"out/soong/.intermediates/defaults/cc/common/libm/android_arm64_armv8-a_shared/libm.so.toc",
		"out/soong/.intermediates/defaults/rust/liblog/android_arm64_armv8-a_shared/liblog.so",
		"out/soong/.intermediates/defaults/rust/liblog/android_arm64_armv8-a_shared/liblog.so.toc",
	}
)

func TestSandboxCompilation(t *testing.T) {
	ctx := testRust(t, `
		filegroup {
			name: "libsrcs1",
			srcs: ["src_filegroup1.rs"],
		}
		filegroup {
			name: "libsrcs2",
			srcs: ["src_filegroup2.rs"],
		}
		rust_library {
			name: "libfizz_buzz",
			crate_name:"fizz_buzz",
			crate_root: "foo.rs",
			srcs: [
				"src_lib*.rs",
				":libsrcs1",
				":libsrcs2",
			],
			compile_data: [
				"compile_data1.txt",
				"compile_data2.txt",
			],
			dylib: {
				srcs: ["dylib_only.rs"],
			},
			rlib: {
				srcs: ["rlib_only.rs"],
			},
		}
		rust_binary {
			name: "fizz_buzz",
			crate_name:"fizz_buzz",
			crate_root: "foo.rs",
			srcs: [
				"src_lib*.rs",
				":libsrcs1",
				":libsrcs2",
			],
		}
		rust_ffi {
			name: "librust_ffi",
			crate_name: "rust_ffi",
			crate_root: "foo.rs",
			static: {
				srcs: ["static_only.rs"],
			},
			shared: {
				srcs: ["shared_only.rs"],
			},
			srcs: ["src1.rs"],
		}
		cc_library_static {
			name: "cc_dep_static",
		}
		cc_library_shared {
			name: "cc_dep_shared",
		}
		rust_library {
			name: "libfizz_buzz_cc_deps",
			crate_name:"fizz_buzz",
			crate_root: "foo.rs",
			srcs: ["src*.rs"],
			shared_libs: ["cc_dep_shared"],
			static_libs: ["cc_dep_static"],
		}
		rust_library {
			name: "libfizz_buzz_intermediate_cc_deps",
			crate_name:"fizz_buzz",
			crate_root: "foo.rs",
			srcs: ["src*.rs"],
			rustlibs: ["libfizz_buzz_cc_deps"],
		}
		rust_library {
			name: "libfizz_buzz_transitive_cc_deps",
			crate_name:"fizz_buzz",
			crate_root: "foo.rs",
			srcs: ["src*.rs"],
			rustlibs: ["libfizz_buzz_intermediate_cc_deps"],
		}
	`,
		android.FixtureMergeMockFs(android.MockFS{
			"src_lib1.rs":       nil,
			"src_lib2.rs":       nil,
			"src_lib3.rs":       nil,
			"src_lib4.rs":       nil,
			"src_filegroup1.rs": nil,
			"src_filegroup2.rs": nil,
			"static_only.rs":    nil,
			"shared_only.rs":    nil,
		}),
	)

	testcases := []struct {
		name                     string
		moduleName               string
		variant                  string
		rustcExpectedFilesToCopy []string
		expectedFlags            []string
	}{
		{
			name:       "rust_library (dylib)",
			moduleName: "libfizz_buzz",
			variant:    "android_arm64_armv8-a_dylib",
			rustcExpectedFilesToCopy: android.Concat(sboxCompilationFiles, sboxCompilationFilesWithCc, []string{
				"foo.rs",
				"src_lib1.rs",
				"src_lib2.rs",
				"src_lib3.rs",
				"src_lib4.rs",
				"src_filegroup1.rs",
				"src_filegroup2.rs",
				"compile_data1.txt",
				"compile_data2.txt",
				"dylib_only.rs",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/out/src_filegroup1.rs",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/out/src_filegroup2.rs",

				"out/soong/.intermediates/defaults/cc/common/libc/android_arm64_armv8-a_shared/libc.so",
				"out/soong/.intermediates/defaults/cc/common/libc/android_arm64_armv8-a_shared/libc.so.toc",
				"out/soong/.intermediates/defaults/cc/common/libm/android_arm64_armv8-a_shared/libm.so",
				"out/soong/.intermediates/defaults/cc/common/libm/android_arm64_armv8-a_shared/libm.so.toc",
				"out/soong/.intermediates/defaults/cc/common/libdl/android_arm64_armv8-a_shared/libdl.so",
				"out/soong/.intermediates/defaults/cc/common/libdl/android_arm64_armv8-a_shared/libdl.so.toc",
				"out/soong/.intermediates/defaults/rust/libstd/android_arm64_armv8-a_dylib/unstripped/libstd.dylib.so",
				"out/soong/.intermediates/defaults/cc/common/crtbegin_so/android_arm64_armv8-a/crtbegin_so.o",
				"out/soong/.intermediates/defaults/cc/common/crtend_so/android_arm64_armv8-a/crtend_so.o",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_dylib/libfizz_buzz.dylib.so.clippy",
			}),
			expectedFlags: []string{
				"-C linker=build/soong/scripts/mkcratersp.py",
				"--emit link",
				"-o __SBOX_SANDBOX_DIR__/out/libfizz_buzz.dylib.so.rsp",
				"--emit dep-info=__SBOX_SANDBOX_DIR__/out/libfizz_buzz.dylib.so.rsp.d.raw",
				"foo.rs", // this is the entry point
			},
		},
		{
			name:       "rust_library (rlib dylib-std)",
			moduleName: "libfizz_buzz",
			variant:    "android_arm64_armv8-a_rlib_dylib-std",
			rustcExpectedFilesToCopy: android.Concat(sboxCompilationFiles, []string{
				"foo.rs",
				"src_lib1.rs",
				"src_lib2.rs",
				"src_lib3.rs",
				"src_lib4.rs",
				"src_filegroup1.rs",
				"src_filegroup2.rs",
				"rlib_only.rs",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_dylib-std/out/src_filegroup1.rs",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_dylib-std/out/src_filegroup2.rs",
				"out/soong/.intermediates/defaults/rust/libstd/android_arm64_armv8-a_dylib/unstripped/libstd.dylib.so",
			}),
			expectedFlags: []string{
				"--emit link",
				"-o __SBOX_SANDBOX_DIR__/out/libfizz_buzz.rlib",
				"--emit dep-info=__SBOX_SANDBOX_DIR__/out/libfizz_buzz.rlib.d.raw",
				"foo.rs", // this is the entry point
			},
		},
		{
			name:       "rust_library (rlib rlib-std)",
			moduleName: "libfizz_buzz",
			variant:    "android_arm64_armv8-a_rlib_rlib-std",
			rustcExpectedFilesToCopy: android.Concat(sboxCompilationFiles, []string{
				"foo.rs",
				"src_lib1.rs",
				"src_lib2.rs",
				"src_lib3.rs",
				"src_lib4.rs",
				"src_filegroup1.rs",
				"src_filegroup2.rs",
				"rlib_only.rs",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/out/src_filegroup1.rs",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/out/src_filegroup2.rs",
				"out/soong/.intermediates/libfizz_buzz/android_arm64_armv8-a_rlib_rlib-std/libfizz_buzz.rlib.clippy",
				"out/soong/.intermediates/defaults/rust/libstd/android_arm64_armv8-a_rlib/libstd.rlib",
			}),
			expectedFlags: []string{
				"--emit link",
				"-o __SBOX_SANDBOX_DIR__/out/libfizz_buzz.rlib",
				"--emit dep-info=__SBOX_SANDBOX_DIR__/out/libfizz_buzz.rlib.d.raw",
				"foo.rs", // this is the entry point
			},
		},
		{
			name:       "rust_binary",
			moduleName: "fizz_buzz",
			variant:    "android_arm64_armv8-a",
			rustcExpectedFilesToCopy: android.Concat(sboxCompilationFiles, sboxCompilationFilesWithCc, []string{
				"foo.rs",
				"src_lib1.rs",
				"src_lib2.rs",
				"src_lib3.rs",
				"src_lib4.rs",
				"src_filegroup1.rs",
				"src_filegroup2.rs",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/out/src_filegroup1.rs",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/out/src_filegroup2.rs",

				"out/soong/.intermediates/defaults/rust/libstd/android_arm64_armv8-a_dylib/unstripped/libstd.dylib.so",
				"out/soong/.intermediates/defaults/cc/common/crtbegin_dynamic/android_arm64_armv8-a/crtbegin_dynamic.o",
				"out/soong/.intermediates/defaults/cc/common/crtend_android/android_arm64_armv8-a/crtend_android.o",
				"out/soong/.intermediates/fizz_buzz/android_arm64_armv8-a/fizz_buzz.clippy",
			}),
			expectedFlags: []string{
				"--emit link",
				"-o __SBOX_SANDBOX_DIR__/out/fizz_buzz",
				"--emit dep-info=__SBOX_SANDBOX_DIR__/out/fizz_buzz.rsp.d.raw",
				"foo.rs", // this is the entry point
			},
		},
		{
			name:       "rust_ffi static lib variant",
			moduleName: "librust_ffi",
			variant:    "android_arm64_armv8-a_static",
			rustcExpectedFilesToCopy: android.Concat(sboxCompilationFiles, []string{
				"foo.rs",
				"src1.rs",
				"static_only.rs",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_static/librust_ffi.a.clippy",
				"out/soong/.intermediates/defaults/rust/libstd/android_arm64_armv8-a_rlib/libstd.rlib",
			}),
			expectedFlags: []string{
				"--emit link",
				"-o __SBOX_SANDBOX_DIR__/out/librust_ffi.a",
				"--emit dep-info=__SBOX_SANDBOX_DIR__/out/librust_ffi.a.d.raw",
				"foo.rs", // this is the entry point
			},
		},
		{
			name:       "rust_ffi shared lib variant",
			moduleName: "librust_ffi",
			variant:    "android_arm64_armv8-a_shared",
			rustcExpectedFilesToCopy: android.Concat(sboxCompilationFiles, sboxCompilationFilesWithCc, []string{
				"foo.rs",
				"src1.rs",
				"shared_only.rs",

				"out/soong/.intermediates/defaults/rust/libstd/android_arm64_armv8-a_dylib/unstripped/libstd.dylib.so",
				"out/soong/.intermediates/defaults/cc/common/crtbegin_so/android_arm64_armv8-a/crtbegin_so.o",
				"out/soong/.intermediates/defaults/cc/common/crtend_so/android_arm64_armv8-a/crtend_so.o",
				"out/soong/.intermediates/librust_ffi/android_arm64_armv8-a_shared/librust_ffi.so.clippy",
			}),
			expectedFlags: []string{
				"--emit link",
				"-o __SBOX_SANDBOX_DIR__/out/librust_ffi.so",
				"--emit dep-info=__SBOX_SANDBOX_DIR__/out/librust_ffi.so.rsp.d.raw",
				"foo.rs", // this is the entry point
			},
		},
		{
			name:       "rust_library with cc deps (dylib)",
			moduleName: "libfizz_buzz_cc_deps",
			variant:    "android_arm64_armv8-a_dylib",
			rustcExpectedFilesToCopy: []string{
				"out/soong/.intermediates/cc_dep_static/android_arm64_armv8-a_static/cc_dep_static.a",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so.toc",
			},
		},
		{
			name:       "rust_library with cc deps (rlib rlib-std)",
			moduleName: "libfizz_buzz_cc_deps",
			variant:    "android_arm64_armv8-a_rlib_rlib-std",
			rustcExpectedFilesToCopy: []string{
				"out/soong/.intermediates/cc_dep_static/android_arm64_armv8-a_static/cc_dep_static.a",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so.toc",
			},
		},
		{
			name:       "rust_library with cc deps (rlib dylib-std)",
			moduleName: "libfizz_buzz_cc_deps",
			variant:    "android_arm64_armv8-a_rlib_dylib-std",
			rustcExpectedFilesToCopy: []string{
				"out/soong/.intermediates/cc_dep_static/android_arm64_armv8-a_static/cc_dep_static.a",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so.toc",
			},
		},
		{
			name:       "rust_library with transitive cc deps (dylib)",
			moduleName: "libfizz_buzz_transitive_cc_deps",
			variant:    "android_arm64_armv8-a_dylib",
			rustcExpectedFilesToCopy: []string{
				"out/soong/.intermediates/cc_dep_static/android_arm64_armv8-a_static/cc_dep_static.a",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so.toc",
			},
		},
		{
			name:       "rust_library with transitive cc deps (rlib rlib-std)",
			moduleName: "libfizz_buzz_transitive_cc_deps",
			variant:    "android_arm64_armv8-a_rlib_rlib-std",
			rustcExpectedFilesToCopy: []string{
				"out/soong/.intermediates/cc_dep_static/android_arm64_armv8-a_static/cc_dep_static.a",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so.toc",
			},
		},
		{
			name:       "rust_library with transitive cc deps (rlib dylib-std)",
			moduleName: "libfizz_buzz_transitive_cc_deps",
			variant:    "android_arm64_armv8-a_rlib_dylib-std",
			rustcExpectedFilesToCopy: []string{
				"out/soong/.intermediates/cc_dep_static/android_arm64_armv8-a_static/cc_dep_static.a",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so",
				"out/soong/.intermediates/cc_dep_shared/android_arm64_armv8-a_shared/cc_dep_shared.so.toc",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			writeFile := ctx.ModuleForTests(tc.moduleName, tc.variant).Rule("unescapedWriteFile")
			contents := writeFile.BuildParams.Args["content"]
			manifestProto := sbox_proto.Manifest{}
			err := prototext.Unmarshal([]byte(contents), &manifestProto)
			if err != nil {
				t.Errorf("expected no errors unmarshaling manifest proto; got %v", err)
			}

			if len(manifestProto.Commands) != 1 {
				t.Errorf("expected 1 command; got %v", len(manifestProto.Commands))
			}

			// check that sandbox contains correct files
			rustc := manifestProto.Commands[0]
			actualFilesToCopy := []string{}
			for _, copy := range rustc.CopyBefore {
				actualFilesToCopy = append(actualFilesToCopy, copy.GetFrom())
			}
			_, expectedFilesNotCopied, _ := android.ListSetDifference(tc.rustcExpectedFilesToCopy, actualFilesToCopy)
			if len(expectedFilesNotCopied) > 0 {
				t.Errorf("did not copy expected files to sbox: %v;\n files copied: %v", expectedFilesNotCopied, actualFilesToCopy)
			}

			rustcCmd := proptools.String(rustc.Command)
			for _, flag := range tc.expectedFlags {
				android.AssertStringDoesContain(
					t,
					fmt.Sprintf(
						"missing flag in rustc invocation; expected to find substring %q; got %q",
						flag,
						rustcCmd,
					),
					rustcCmd,
					flag,
				)
			}
		})
	}
}
