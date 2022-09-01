// Copyright 2022 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package bp2build

import (
	"fmt"
	"testing"

	"android/soong/android"
	"android/soong/cc"
)

func runCcPrebuiltBinaryTests(t *testing.T, testCase Bp2buildTestCase) {
	t.Helper()
	runCcPrebuiltBinaryTestCaseDevice(t, testCase)
}

func runCcPrebuiltBinaryTestCaseDevice(t *testing.T, testCase Bp2buildTestCase) {
	t.Helper()
	runCcPrebuiltBinaryTestCase(t, testCase, "cc_prebuilt_binary", cc.PrebuiltBinaryFactory)
}

func runCcPrebuiltBinaryTestCase(t *testing.T, testCase Bp2buildTestCase, moduleTypeUnderTest string, factory android.ModuleFactory) {
	t.Helper()
	description := fmt.Sprintf("%s: %s", moduleTypeUnderTest, testCase.Description)
	testCase.ModuleTypeUnderTest = moduleTypeUnderTest
	testCase.ModuleTypeUnderTestFactory = factory
	testCase.Description = description
	t.Run(description, func(t *testing.T) {
		t.Helper()
		RunBp2BuildTestCaseSimple(t, testCase)
	})
}

func TestPrebuiltBinary(t *testing.T) {
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: "cc_prebuilt_binary simple",
			Filesystem: map[string]string{
				"bin": "",
			},
			Blueprint: `
cc_prebuilt_binary {
	name: "bintest",
	srcs: ["bin"],
	bazel_module: { bp2build_available: true },
}`,
			ExpectedBazelTargets: []string{
				MakeBazelTarget("cc_prebuilt_binary", "bintest", AttrNameToString{
					"src": `"bin"`,
				})},
		})
}

func TestPrebuiltBinaryWithStrip(t *testing.T) {
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: "with strip",
			Filesystem: map[string]string{
				"bin": "",
			},
			Blueprint: `
cc_prebuilt_binary {
	name: "bintest",
	srcs: ["bin"],
	strip: { all: true },
	bazel_module: { bp2build_available: true },
}`, ExpectedBazelTargets: []string{
				MakeBazelTarget("cc_prebuilt_binary", "bintest", AttrNameToString{
					"src": `"bin"`,
					"strip": `{
        "all": True,
    }`,
				}),
			},
		})
}

func TestPrebuiltBinaryWithArchVariance(t *testing.T) {
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: "with arch variance",
			Filesystem: map[string]string{
				"bina": "",
				"binb": "",
			},
			Blueprint: `
cc_prebuilt_binary {
	name: "bintest",
	arch: {
		arm64: { srcs: ["bina"], },
		arm: { srcs: ["binb"], },
	},
	bazel_module: { bp2build_available: true },
}`, ExpectedBazelTargets: []string{
				MakeBazelTarget("cc_prebuilt_binary", "bintest", AttrNameToString{
					"src": `select({
        "//build/bazel/platforms/arch:arm": "binb",
        "//build/bazel/platforms/arch:arm64": "bina",
        "//conditions:default": None,
    })`,
				}),
			},
		})
}

func TestPrebuiltBinaryMultipleSrcsFails(t *testing.T) {
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: "fails because multiple sources",
			Filesystem: map[string]string{
				"bina": "",
				"binb": "",
			},
			Blueprint: `
cc_prebuilt_binary {
	name: "bintest",
	srcs: ["bina", "binb"],
	bazel_module: { bp2build_available: true },
}`,
			ExpectedErr: fmt.Errorf("Expected at most one source file"),
		})
}
