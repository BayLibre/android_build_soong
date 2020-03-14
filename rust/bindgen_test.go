// Copyright 2020 Google Inc. All rights reserved.
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
	"strings"
	"testing"
)

// Test that bindings generator is built and called to generate rust bindings file
func TestBindingGenerator(t *testing.T) {
	ctx := testRust(t, `
		rust_library_host_rlib {
			name: "libbindgen",
			crate_name: "bindgen",
			srcs: [""],
		}

		rust_bindgen_rlib {
			name: "libfoo_sys",
			crate_name: "foo_sys",
			srcs: ["foo.rs"],
			headers: ["header.h"],
			shared_libs: [
				"libutils",
			],
			host_supported: true,
		}

		// We want to link against a shared library to test include flags
		cc_library {
			name: "libutils",
			no_libcrt: true,
			nocrt: true,
			system_shared_libs: [],
			host_supported: true,
			export_include_dirs: ["include"],
		}`)

	generator := ctx.ModuleForTests("libfoo_sys-bindgen", "linux_glibc_x86_64").Output("libfoo_sys-bindgen")
	if !strings.Contains(generator.Args["libFlags"], "libbindgen.rlib") {
		t.Errorf("Bindings generator tool not linked against the bindgen crate.")
	}

	sysModule := ctx.ModuleForTests("libfoo_sys", "android_arm64_armv8-a_rlib")

	generatorParams := sysModule.Rule("bindingsGenerator")
	if !strings.Contains(generatorParams.Args["generator"], "libfoo_sys-bindgen") {
		t.Errorf("Unexpected bindings generator binary used to generate bindings")
	}
	if generatorParams.Inputs[0].String() != "header.h" {
		t.Errorf("Could not find input header in generator invocation")
	}
	if !strings.Contains(generatorParams.Args["extraFlags"], "-Iinclude") {
		t.Errorf("Did not find dependency include dir in generator invocation")
	}
	if !strings.Contains(generatorParams.Output.String(), "gen/bindings.rs") {
		t.Errorf("Unexpected output file for bindings generator")
	}

	sysModuleBuildParams := sysModule.Output("libfoo_sys.rlib")
	if !strings.Contains(sysModuleBuildParams.Inputs[0].String(), "gen/bindings.rs") {
		t.Errorf("Did not find Rust bindings file in sys module inputs")
	}
	if !strings.Contains(sysModuleBuildParams.Args["rustcFlags"], "-ldylib=utils") {
		t.Errorf("Sys module was not linked against libutils system shared library")
	}

	// Check that we can also generate bindings rlib for host
	ctx.ModuleForTests("libfoo_sys", "linux_glibc_x86_64_rlib").Output("libfoo_sys.rlib")
}
