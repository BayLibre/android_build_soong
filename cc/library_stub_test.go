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
	"strings"
	"testing"

	"android/soong/android"
)

func TestCcApiStubLibraryRequiredProperties(t *testing.T) {
	testCases := []struct {
		bp             string
		expectedErrMsg string
	}{
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
					version: "29",
				}
			`,
			expectedErrMsg: "stem is a required field",
		},
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
					stem: "foo",
					version: "29",
				}
			`,
			expectedErrMsg: "symbol_file is a required field",
		},
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
					stem: "foo",
					symbol_file: "foo.map.txt",
				}
			`,
			expectedErrMsg: "version is a required field",
		},
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
					stem: "foo",
					symbol_file: "foo.map.txt",
					version: "unrecognized_version",
				}
			`,
			expectedErrMsg: "\"unrecognized_version\" could not be parsed as an integer and is not a recognized codename",
		},

		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
					stem: "foo",
					symbol_file: "foo.map.txt",
					version: "29",
				}
			`,
			expectedErrMsg: "",
		},
	}
	for _, testCase := range testCases {
		fixture := prepareForCcTest
		if testCase.expectedErrMsg != "" {
			fixture = fixture.
				ExtendWithErrorHandler(android.FixtureExpectsAtLeastOneErrorMatchingPattern(testCase.expectedErrMsg))
		}
		fixture.RunTestWithBp(t, testCase.bp)
	}

}

func TestCcApiStubLibraryOutputFiles(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo-stubs",
			stem: "foo",
			symbol_file: "foo.map.txt",
			version: "29",
		}
	`
	result := prepareForCcTest.RunTestWithBp(t, bp)
	outputs := result.ModuleForTests("foo-stubs", "android_arm64_armv8-a_shared").AllOutputs()
	expected_file_suffixes := []string{"stub.c", "stub.map", "stub.o", "foo-stubs.so"}
	for _, expected_file_suffix := range expected_file_suffixes {
		android.AssertBoolEquals(t, expected_file_suffix+" file not found in output", true, android.SuffixInList(outputs, expected_file_suffix))
	}
}

func TestCcApiStubLibraryVersionFlag(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo-stubs",
			stem: "foo",
			symbol_file: "foo.map.txt",
			version: "29",
		}
	`
	result := prepareForCcTest.RunTestWithBp(t, bp)
	stubGenArgs := result.ModuleForTests("foo-stubs", "android_arm65_armv8-a_shared").Output("stub.c").Args
	android.AssertStringEquals(t, "Incorrect version used to generate stubs", "29", stubGenArgs["apiLevel"])
}

// TODO: Test stem is required or defaults to name
// TODO: extra args for #llndk #apex

func TestStemModuleCreation(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo_in_bar_api_surface",
			stem: "foo",
			symbol_file: "foo.map.txt",
			version: "29",
		}

		cc_api_stub_library {
			name: "foo_in_baz_api_surface",
			stem: "foo",
			symbol_file: "foo.map.txt",
			version: "29",
		}
	`
	// Assert that the stem exists
	result := prepareForCcTest.RunTestWithBp(t, bp)
	_ = result.ModuleForTests("foo", "android_arm64_armv8-a_shared")
}

func TestCcLibraryUsesCcApiStubLibrary(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo-stubs",
			stem: "foo",
			symbol_file: "foo.map.txt",
			version: "29",
		}
		cc_library {
			name: "systemlib",
			shared_libs: [
				"foo",
			],
		}
		cc_library {
			name: "vendorlib",
			shared_libs: [
				"foo",
			],
			vendor: true,
		}


	`
	moduleIsSharedLibraryDep := func(parent android.TestingModule, childName string) bool {
		argsToLinkRule := parent.Rule("ld").Args["libFlags"]
		childNameWithExt := childName + ".so"
		return strings.Contains(argsToLinkRule, childNameWithExt)
	}
	result := prepareForCcTest.RunTestWithBp(t, bp)
	systemlib := result.ModuleForTests("systemlib", "android_arm64_armv8-a_shared")
	android.AssertBoolEquals(t, "systemlib should link against source", true, moduleIsSharedLibraryDep(systemlib, "foo"))
	android.AssertBoolEquals(t, "systemlib should link not against stubs", false, moduleIsSharedLibraryDep(systemlib, "foo-stubs"))
	vendorlib := result.ModuleForTests("vendorlib", "android_vendor.29_arm64_armv8-a_shared")
	android.AssertBoolEquals(t, "vendorlib should link not against source", false, moduleIsSharedLibraryDep(vendorlib, "foo"))
	android.AssertBoolEquals(t, "vendorlib should link against stubs", true, moduleIsSharedLibraryDep(vendorlib, "foo-stubs"))
}
