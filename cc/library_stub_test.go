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
	"fmt"
	"strings"
	"testing"

	"android/soong/android"
)

var prepareForCcMultiTreeTest = android.GroupFixturePreparers(
	prepareForCcTest,
	android.FixtureModifyEnv(func(env map[string]string) {
		env["MULTI_TREE"] = "true"
	}),
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
				}
			`,
			expectedErrMsg: "symbol_file is a required field",
		},
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
				}
			`,
			expectedErrMsg: "version is a required field",
		},
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
				}
			`,
			expectedErrMsg: "api_surface is a required field",
		},
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
					symbol_file: "foo.map.txt",
					version: "29",
					api_surface: "invalid_api_surface",
				}
			`,
			expectedErrMsg: "invalid_api_surface is not a recognized API Surface",
		},
		{
			bp: `
				cc_api_stub_library {
					name: "foo-stubs",
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
					symbol_file: "foo.map.txt",
					version: "29",
					api_surface: "public",
				}
			`,
			expectedErrMsg: "",
		},
	}
	for _, testCase := range testCases {
		fixture := prepareForCcMultiTreeTest
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
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "public",
			version: "29",
		}
	`
	result := prepareForCcMultiTreeTest.RunTestWithBp(t, bp)
	outputs := result.ModuleForTests("foo.public.29", "android_arm64_armv8-a_shared").AllOutputs()
	expected_file_suffixes := []string{"stub.c", "stub.map", "stub.o", "foo.so"}
	for _, expected_file_suffix := range expected_file_suffixes {
		android.AssertBoolEquals(t, expected_file_suffix+" file not found in output", true, android.SuffixInList(outputs, expected_file_suffix))
	}
}

func TestCcApiStubLibraryVersionFlag(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "vendor",
			version: "29",
		}
	`
	result := prepareForCcMultiTreeTest.RunTestWithBp(t, bp)
	stubGenArgs := result.ModuleForTests("foo.vendor.29", "android_arm64_armv8-a_shared").Output("stub.c").Args
	android.AssertStringEquals(t, "Incorrect version used to generate stubs", "29", stubGenArgs["apiLevel"])
}

func TestCcApiStubLibraryApiSurfaceFlag(t *testing.T) {
	bpTemplate := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "%v",
			version: "29",
		}
	`
	testCases := []struct {
		desc                    string
		apiSurfaceName          string
		expectedNdkStubGenFlags string
	}{
		{
			desc:                    "Stubgen for NDK libraries should not contain any additional args",
			apiSurfaceName:          android.PublicApi.String(),
			expectedNdkStubGenFlags: "",
		},
		{
			desc:                    "Stubgen for libraries presented to vendor should pass --llndk to ndkstubgen invocation",
			apiSurfaceName:          android.VendorApi.String(),
			expectedNdkStubGenFlags: "--llndk",
		},
		{
			desc:                    "Stubgen for libraries presented to apex should pass --apex to ndkstubgen invocation",
			apiSurfaceName:          android.SystemApi.String(),
			expectedNdkStubGenFlags: "--apex",
		},
	}
	for _, testCase := range testCases {
		result := prepareForCcMultiTreeTest.RunTestWithBp(t, fmt.Sprintf(bpTemplate, testCase.apiSurfaceName))
		moduleName := "foo." + testCase.apiSurfaceName + ".29"
		stubGenArgs := result.ModuleForTests(moduleName, "android_arm64_armv8-a_shared").Output("stub.c").Args
		android.AssertStringEquals(t, testCase.desc, testCase.expectedNdkStubGenFlags, stubGenArgs["flags"])
	}
}

// TODO: Test stem is required or defaults to name
// TODO: extra args for #llndk #apex
// TODO: Test apex symbols are not visible
func TestCcLibraryUsesCcApiStubLibrary(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "vendor",
			version: "29",
		}
		//base library
		cc_library {
			name: "foo",
			vendor_available: true,
		}
		cc_library {
			name: "systemlib",
			shared_libs: [
				"foo"
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
		return strings.Contains(argsToLinkRule, "out/soong/.intermediates/"+childName+"/")
	}
	result := prepareForCcMultiTreeTest.RunTestWithBp(t, bp)
	systemlib := result.ModuleForTests("systemlib", "android_arm64_armv8-a_shared")
	android.AssertBoolEquals(t, "systemlib should link against source", true, moduleIsSharedLibraryDep(systemlib, "foo"))
	android.AssertBoolEquals(t, "systemlib should link not against stubs", false, moduleIsSharedLibraryDep(systemlib, "foo.vendor.29"))
	vendorlib := result.ModuleForTests("vendorlib", "android_vendor.29_arm64_armv8-a_shared")
	android.AssertBoolEquals(t, "vendorlib should link not against source", false, moduleIsSharedLibraryDep(vendorlib, "foo"))
	android.AssertBoolEquals(t, "vendorlib should link against stubs", true, moduleIsSharedLibraryDep(vendorlib, "foo.vendor.29"))
}

func TestCcLibraryUsesVersionedCcApiStubLibrary(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "vendor",
			version: "28",
		}
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "vendor",
			version: "29",
		}
		//base library
		cc_library {
			name: "foo",
			vendor_available: true,
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
		return strings.Contains(argsToLinkRule, "out/soong/.intermediates/"+childName+"/")
	}
	result := prepareForCcMultiTreeTest.RunTestWithBp(t, bp)
	vendorlib := result.ModuleForTests("vendorlib", "android_vendor.29_arm64_armv8-a_shared")
	android.AssertBoolEquals(t, "vendorlib should link against stub library with platform_vndk_version", true, moduleIsSharedLibraryDep(vendorlib, "foo.vendor.29"))
	android.AssertBoolEquals(t, "vendorlib should link against stub library with platform_vndk_version", false, moduleIsSharedLibraryDep(vendorlib, "foo.vendor.28"))
}

// Libraries that set `sdk_version` build against the Soong module libc.ndk and not stubs of the Soong module libc
func TestSdkCcLibraryUsesNdkCcApiStubLibrary(t *testing.T) {
	bp := `
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "public",
			version: "29",
		}
		cc_api_stub_library {
			name: "foo",
			symbol_file: "foo.map.txt",
			api_surface: "vendor",
			version: "29",
		}
		//base library
		cc_library {
			name: "foo",
			vendor_available: true,
		}
		//base ndk library
		ndk_library {
			name: "foo",
			first_version: "minimum",
			symbol_file: "foo.map.txt"
		}
		cc_library {
			name: "vendorlib",
			shared_libs: [
				"foo",
			],
			sdk_version: "29",
			stl: "libc++",
			vendor: true,
		}
	`
	moduleIsSharedLibraryDep := func(parent android.TestingModule, childName string) bool {
		argsToLinkRule := parent.Rule("ld").Args["libFlags"]
		return strings.Contains(argsToLinkRule, "out/soong/.intermediates/"+childName+"/")
	}
	result := prepareForCcMultiTreeTest.RunTestWithBp(t, bp)
	vendorlib := result.ModuleForTests("vendorlib", "android_arm64_armv8-a_sdk_shared")
	android.AssertBoolEquals(t, "vendorlib building with sdk should link against ndk stubs", true, moduleIsSharedLibraryDep(vendorlib, "foo.public.29"))
	android.AssertBoolEquals(t, "vendorlib building with sdk should not link against llndk stubs", false, moduleIsSharedLibraryDep(vendorlib, "foo.vendor.29"))
	android.AssertBoolEquals(t, "vendorlib building with sdk should not link against source", false, moduleIsSharedLibraryDep(vendorlib, "foo"))
}
