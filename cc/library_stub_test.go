// Copyright 2021 Google Inc. All rights reserved.
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
	"testing"

	"android/soong/android"
)

func TestCcApiStubLibraryOutputFiles(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
	`
	result := prepareForCcTest.RunTestWithBp(t, bp)
	outputs := result.ModuleForTests("foo", "android_arm64_armv8-a_shared").AllOutputs()
	expected_file_suffixes := []string{".c", "stub.map", ".o", ".so"}
	for _, expected_file_suffix := range expected_file_suffixes {
		android.AssertBoolEquals(t, expected_file_suffix+" file not found in output", true, android.SuffixInList(outputs, expected_file_suffix))
	}
}

func TestCcApiStubLibraryVariants(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
	`
	result := prepareForCcTest.RunTestWithBp(t, bp)
	variants := result.ModuleVariantsForTests("foo")
	expected_variants := []string{"29", "30", "S", "Tiramisu"} //TODO: make this test deterministic by using fixtures
	for _, expected_variant := range expected_variants {
		android.AssertBoolEquals(t, expected_variant+" variant not found in foo", true, android.SubstringInList(variants, expected_variant))
	}
}

func TestCcLibraryUsesCcApiStubLibrary(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			first_version: "29",
		}
		cc_library {
			name: "foo_user",
			shared_libs: [
				"foo#29",
			],
		}

	`
	prepareForCcTest.RunTestWithBp(t, bp)
}
