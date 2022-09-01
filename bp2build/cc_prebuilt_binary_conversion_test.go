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
	"strings"
	"testing"
	"unicode"

	"android/soong/android"
	"android/soong/cc"
)

var prebuiltBinaryReplacer = strings.NewReplacer(ccBinaryTypePlaceHolder, "cc_prebuilt_binary")
var hostPrebuiltBinaryReplacer = strings.NewReplacer(ccBinaryTypePlaceHolder, "cc_prebuilt_binary_host")

func runCcPrebuiltBinaryTests(t *testing.T, testCase Bp2buildTestCase) {
	t.Helper()
	runCcPrebuiltBinaryTestCase(t, testCase, "cc_prebuilt_binary", cc.PrebuiltBinaryFactory, prebuiltBinaryReplacer)
	// FIXME(alexmarquez): Resolve:
	/* target_compatible_with = select({
	    "//build/bazel/platforms/os:android": ["@platforms//:incompatible"],
	    "//conditions:default": [],
	})*/
	// ... being in all host cases
	//runCcPrebuiltBinaryTestCase(t, testCase, "cc_prebuilt_binary_host", cc.PrebuiltBinaryHostFactory, hostPrebuiltBinaryReplacer)
}

func runCcPrebuiltBinaryTestCase(t *testing.T, testCase Bp2buildTestCase, moduleTypeUnderTest string, factory android.ModuleFactory, blueprintReplacer *strings.Replacer) {
	t.Helper()
	description := fmt.Sprintf("%s: %s", moduleTypeUnderTest, testCase.Description)
	t.Run(description, func(t *testing.T) {
		t.Helper()
		testCase.ModuleTypeUnderTest = moduleTypeUnderTest
		testCase.ModuleTypeUnderTestFactory = factory
		testCase.Description = description
		testCase.Blueprint = blueprintReplacer.Replace(testCase.Blueprint)
		RunBp2BuildTestCaseSimple(t, testCase)
	})
}

func TestPrebuiltBinary(t *testing.T) {
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: "simple",
			Filesystem: map[string]string{
				"bin": "",
			},
			Blueprint: `
{rule_name} {
	name: "bintest",
	srcs: ["bin"],
	bazel_module: { bp2build_available: true },
}`,
			ExpectedBazelTargets: []string{
				MakeBazelTarget("prebuilt_binary", "bintest", AttrNameToString{
					"binary":          `"bin"`,
					"check_elf_files": "True",
				}),
			},
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
{rule_name} {
	name: "bintest",
	srcs: ["bin"],
	strip: { all: true },
	bazel_module: { bp2build_available: true },
}`,
			ExpectedBazelTargets: []string{
				MakeBazelTarget("prebuilt_binary", "bintest", AttrNameToString{
					"binary": `"bin"`,
					"strip": `{
        "all": True,
    }`,
					"check_elf_files": "True",
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
{rule_name} {
	name: "bintest",
	arch: {
		arm64: { srcs: ["bina"], },
		arm: { srcs: ["binb"], },
	},
	bazel_module: { bp2build_available: true },
}`,
			ExpectedBazelTargets: []string{
				MakeBazelTarget("prebuilt_binary", "bintest", AttrNameToString{
					"binary": `select({
        "//build/bazel/platforms/arch:arm": "binb",
        "//build/bazel/platforms/arch:arm64": "bina",
        "//conditions:default": None,
    })`,
					"check_elf_files": "True",
				}),
			},
		})
}

/*func TestPrebuiltBinaryNoSrcsFails(t *testing.T) {
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: "fails because no sources",
			Filesystem:  map[string]string{},
			Blueprint: `
{rule_name} {
	name: "bintest",
	srcs: [],
	bazel_module: { bp2build_available: true },
}`,
			ExpectedErr: fmt.Errorf("Expected at most one source file"),
		})
}*/

func TestPrebuiltBinaryMultipleSrcsFails(t *testing.T) {
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: "fails because multiple sources",
			Filesystem: map[string]string{
				"bina": "",
				"binb": "",
			},
			Blueprint: `
{rule_name} {
	name: "bintest",
	srcs: ["bina", "binb"],
	bazel_module: { bp2build_available: true },
}`,
			ExpectedErr: fmt.Errorf("Expected at most one source file"),
		})
}

func testPrebuiltBinaryCheckElf(t *testing.T, checkElfValue string) {
	t.Helper()
	soongCheckElfValue := checkElfValue
	a := []rune(soongCheckElfValue)
	a[0] = unicode.ToLower(a[0])
	soongCheckElfValue = string(a)
	runCcPrebuiltBinaryTests(t,
		Bp2buildTestCase{
			Description: fmt.Sprintf("prebuilt binary with check_elf %s", checkElfValue),
			Filesystem: map[string]string{
				"bin": "",
			},
			Blueprint: fmt.Sprintf(`
cc_prebuilt_binary {
	name: "bintest",
	srcs: ["bin"],
	check_elf_files: %s,
	bazel_module: { bp2build_available: true },
}`, soongCheckElfValue),
			ExpectedBazelTargets: []string{
				MakeBazelTarget("prebuilt_binary", "bintest", AttrNameToString{
					"binary":          `"bin"`,
					"check_elf_files": fmt.Sprintf(`%s`, checkElfValue),
				}),
			},
		})
}

func TestPrebuiltBinaryCheckElfTrue(t *testing.T) {
	testPrebuiltBinaryCheckElf(t, "True")
}

func TestPrebuiltBinaryCheckElfFalse(t *testing.T) {
	testPrebuiltBinaryCheckElf(t, "False")
}

// TODO: Check ELF for unset
// Is this necessary given e.g. the Simple case already has it unset?
//func TestPrebuiltBinaryCheckElf()
