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

package cc

import (
	_ "fmt"
	"testing"

	"android/soong/android"
)

func TestCcApiStubLibraryOutputFiles(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			api_surface_name: "ndk",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
	`
	result := prepareForCcTest.RunTestWithBp(t, bp)
	outputs := result.ModuleForTests("foo.ndk.import", "android_arm64_armv8-a_shared_current").AllOutputs()
	expected_file_suffixes := []string{".c", "stub.map", ".o", ".so"}
	for _, expected_file_suffix := range expected_file_suffixes {
		android.AssertBoolEquals(t, expected_file_suffix+" file not found in output", true, android.SuffixInList(outputs, expected_file_suffix))
	}
}

func TestCcApiStubLibraryVariants(t *testing.T) {
	pinnedCodenames := android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
		variables.Platform_version_active_codenames = []string{"S", "Tiramisu"}
	})
	bp := `
		cc_api_stub_library {
			name: "foo",
			api_surface_name: "ndk",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
	`
	fixture := android.GroupFixturePreparers(
		prepareForCcTest,
		pinnedCodenames,
	)
	result := fixture.RunTestWithBp(t, bp)
	variants := result.ModuleVariantsForTests("foo.ndk.import")
	expected_variants := []string{"29", "30", "S", "Tiramisu", "current"}
	for _, expected_variant := range expected_variants {
		android.AssertBoolEquals(t, expected_variant+" variant not found in foo", true, android.SubstringInList(variants, expected_variant))
	}
}

// check stub library does not cause a name collision with an ndk_library and a cc_library with the same name
func TestCcApiStubLibraryNameCollision(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			api_surface_name: "ndk",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
		ndk_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
		// foo impl
		cc_library {
			name: "foo",
		}
	`
	prepareForCcTest.RunTestWithBp(t, bp)
}

// TODO: checkdeps
func TestCcLibraryUsesCcApiStubLibrary(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			api_surface_name: "ndk",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
		cc_library {
			name: "foo_user",
			shared_libs: [
				"foo.ndk.import#29",
			],
		}

	`
	prepareForCcTest.RunTestWithBp(t, bp)
}
