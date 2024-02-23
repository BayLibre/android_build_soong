// Copyright 2024 The Android Open Source Project
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
	"android/soong/cc"
	"strings"
	"testing"
)

func TestGeneratedLibrary(t *testing.T) {
	ctx := testRust(t, `
		cc_binary {
			name: "fizzbuzz",
			static_rlibs: ["libfoo.ffi"],
			host_supported: true,
		}
		cc_library {
			name: "libcc",
			static_rlibs: ["libfoo.ffi"],
			host_supported: true,
		}
		rust_ffi_rlib {
			name: "libfoo.ffi",
			srcs: ["foo.rs"],
			rustlibs: ["libbar_rs"],
			crate_name: "foo",
			host_supported: true,
		}
		rust_library {
			name: "libbar_rs",
			srcs: ["foo.rs"],
			crate_name: "bar_rs",
			host_supported: true,
		}
	`)

	ccBinModule := ctx.ModuleForTests("fizzbuzz", "linux_glibc_x86_64").Module().(*cc.Module)
	ccLibModule := ctx.ModuleForTests("libcc", "linux_glibc_x86_64_shared").Module().(*cc.Module)
	genCcBinRule := ctx.ModuleForTests("libfizzbuzz_generated_rust_staticlib", "linux_glibc_x86_64_static_source").Rule("rustc")

	// Check the crate type
	staticCrateType := "staticlib"
	if !strings.Contains(genCcBinRule.Args["rustcFlags"], "crate-type="+staticCrateType) {
		t.Errorf("incorrect crate-type for generated_library static variant, expecting crate-type=%#v, rustcFlags: %#v", staticCrateType, genCcBinRule.Args["rustcFlags"])
	}

	// Make sure cc_binaries are generating static libs appropriately.
	if !android.InList("libfizzbuzz_generated_rust_staticlib", ccBinModule.Properties.AndroidMkStaticLibs) {
		t.Errorf("libfizzbuzz_generated_rust_staticlib expected to be a dependency of cc_binary static libraries. Static lib deps are: %#v",
			ccBinModule.Properties.AndroidMkStaticLibs)
	}

	// Make sure cc_libraries are generating static libs appropriately.
	if !android.InList("liblibcc_generated_rust_staticlib", ccLibModule.Properties.AndroidMkStaticLibs) {
		t.Errorf("liblibcc_generated_rust_staticlib expected to be a dependency of cc_library static libraries. Static lib deps are: %#v",
			ccLibModule.Properties.AndroidMkStaticLibs)
	}
}

func TestGeneratedLibraryDeps(t *testing.T) {
	// Check deps propagate as expected.
	// E.g. all rlibs are included in the generated library
	// and the generated library is passed upwards to all dependants
	ctx := testRust(t, `
		cc_binary {
			name: "fizzbuzz",
			srcs: ["foo.c"],
			static_libs: ["libaa"],
			host_supported: true,
		}
		cc_library {
			name: "libaa",
			srcs: ["foo.c"],
			static_libs: [
				"libcc",
			],
			host_supported: true,
		}
		cc_library {
			name: "libcc",
			srcs: ["foo.c"],
			static_rlibs: [
				"libfoo.ffi",
			],
			host_supported: true,
		}
		rust_ffi_rlib {
			name: "libfoo.ffi",
			srcs: ["foo.rs"],
			rustlibs: ["libfoo_rs"],
			crate_name: "foo",
			host_supported: true,
		}
		rust_library {
			name: "libbar_rs",
			srcs: ["bar.rs"],
			crate_name: "bar_rs",
			host_supported: true,
		}
		rust_library {
			name: "libfoo_rs",
			srcs: ["foo.rs"],
			crate_name: "foo_rs",
			rustlibs: ["libbar_rs"],
			host_supported: true,
		}
	`)

	genCcLib := ctx.ModuleForTests("liblibcc_generated_rust_staticlib", "linux_glibc_x86_64_static_source").Module().(*Module)
	genCcRule := ctx.ModuleForTests("liblibcc_generated_rust_staticlib", "linux_glibc_x86_64_static_source").Rule("rustc")
	ccBinRule := ctx.ModuleForTests("fizzbuzz", "linux_glibc_x86_64").Rule("ld")

	// rlibs propagate through Soong Rust via -L flags
	// liblibcc_generated_rust_staticlib links against libfoo_rs, indirect dependency through libfoo.ffi
	if !strings.Contains(genCcRule.Args["libFlags"], "-L out/soong/.intermediates/libbar_rs/linux_glibc_x86_64_rlib_rlib-std/") {
		t.Errorf("expected  '-L out/soong/.intermediates/libbar_rs/linux_glibc_x86_64_rlib_rlib-std/' in linkFlags, got: %#v",
			genCcRule.Args["libFlags"])
	}

	// rlibs propagate through Soong Rust via -L flags
	// liblibcc_generated_rust_staticlib links against libbar_rs, indirect dependency through libfoo_rs
	if !strings.Contains(genCcRule.Args["libFlags"], "-L out/soong/.intermediates/libfoo_rs/linux_glibc_x86_64_rlib_rlib-std/") {
		t.Errorf("expected  '-L out/soong/.intermediates/libbar_rs/linux_glibc_x86_64_rlib_rlib-std/' in linkFlags, got: %#v",
			genCcRule.Args["libFlags"])
	}

	// test that the static_rlibs modules are direct dependencies of the generated library.
	if !android.InList("libfoo.ffi.rlib-std", genCcLib.Properties.AndroidMkRlibs) {
		t.Errorf("libfoo.ffi.rlib-std expected to be a dependency of liblibcc_generated_rust_staticlib. Rlib deps are: %#v",
			genCcLib.Properties.AndroidMkRlibs)
	}

	// Check that generated libraries propagate through to dependents further down the dependency tree.
	if !strings.Contains(ccBinRule.Args["libFlags"], "liblibcc_generated_rust_staticlib") {
		t.Errorf("expected  'liblibcc_generated_rust_staticlib' in linkFlags, got: %#v",
			ccBinRule.Args["libFlags"])
	}
}

func TestGeneratedLibraryIncludes(t *testing.T) {
	// Test that includes are being exported
	ctx := testRust(t, `
		cc_binary {
			name: "fizzbuzz",
			srcs: ["foo.c"],
			static_libs: ["libcc"],
			host_supported: true,
		}
		cc_library {
			name: "libcc",
			srcs: ["foo.c"],
			static_rlibs: [
				"libfoo.ffi",
				"libbar.ffi"
			],
			host_supported: true,
		}
		rust_ffi_rlib {
			name: "libfoo.ffi",
			srcs: ["foo.rs"],
			rustlibs: ["libfoo_rs"],
			crate_name: "foo",
			host_supported: true,
			include_dirs: ["include/foo"]
		}
		rust_ffi_rlib {
			name: "libbar.ffi",
			srcs: ["bar.rs"],
			rustlibs: ["libfoo_rs"],
			crate_name: "bar",
			host_supported: true,
			include_dirs: ["include/bar"]
		}
		rust_library {
			name: "libfoo_rs",
			srcs: ["foo.rs"],
			crate_name: "foo_rs",
			host_supported: true,
		}
		rust_library {
			name: "libbar_rs",
			srcs: ["bar.rs"],
			crate_name: "bar_rs",
			host_supported: true,
		}
	`)

	ccBin := ctx.ModuleForTests("fizzbuzz", "linux_glibc_x86_64").Rule("cc")
	ccLib := ctx.ModuleForTests("libcc", "linux_glibc_x86_64_static").Rule("cc")

	// Make sure the generated library is exporting to direct dependents its include dirs.
	if !strings.Contains(ccLib.Args["cFlags"], "-Iinclude/bar") {
		t.Errorf("expected  '-Iinclude/bar' in cFlags, got: %#v",
			ccLib.Args["cFlags"])
	}
	if !strings.Contains(ccLib.Args["cFlags"], "-Iinclude/foo") {
		t.Errorf("expected  '-Iinclude/foo' in cFlags, got: %#v",
			ccLib.Args["cFlags"])
	}

	// Make sure that cc_library is re-exporting the generated static librarys exported dirs.
	if !strings.Contains(ccBin.Args["cFlags"], "-Iinclude/foo") {
		t.Errorf("expected  '-Iinclude/foo' in cFlags, got: %#v",
			ccBin.Args["cFlags"])
	}
	if !strings.Contains(ccBin.Args["cFlags"], "-Iinclude/bar") {
		t.Errorf("expected  '-Iinclude/bar' in cFlags, got: %#v",
			ccBin.Args["cFlags"])
	}
}

func TestGeneratedLibraryStdConfig(t *testing.T) {
	// Test that includes are being exported
	ctx := testRust(t, `
		cc_library {
			name: "libcc",
			srcs: ["foo.c"],
			static_rlibs: [
				"libfoo.ffi",
			],
			static_rlibs_config: {
				no_stdlibs: true,
				stdlibs: ["libcore.sysroot"],
			},
			host_supported: true,
		}
		rust_ffi_rlib {
			name: "libfoo.ffi",
			srcs: ["foo.rs"],
			crate_name: "foo",
			no_stdlibs: true,
			stdlibs: ["libcore.sysroot"],
			host_supported: true,
			include_dirs: ["include/foo"]
		}
	`)

	genLibCompiler := ctx.ModuleForTests("liblibcc_generated_rust_staticlib", "linux_glibc_x86_64_static_source").Module().(*Module).compiler.(*libraryDecorator).baseCompiler

	if !android.InList("libcore.sysroot", genLibCompiler.Properties.Stdlibs) {
		t.Errorf("libcore.sysroot expected to be a stdlib dependency of liblibcc_generated_rust_staticlib. Stdlib deps are: %#v",
			genLibCompiler.Properties.Stdlibs)
	}
	if !Bool(genLibCompiler.Properties.No_stdlibs) {
		t.Errorf("no_stdlibs expected to be a set for liblibcc_generated_rust_staticlib, instead set to false.")
	}
}
