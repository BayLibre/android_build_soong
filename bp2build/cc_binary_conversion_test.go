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

package bp2build

import (
	"android/soong/cc"
	"fmt"
	"strings"
	"testing"
)

const (
	ccBinaryTypePlaceHolder   = "{rule_name}"
	compatibleWithPlaceHolder = "{target_compatible_with}"
)

var binaryReplacer = strings.NewReplacer(ccBinaryTypePlaceHolder, "cc_binary", compatibleWithPlaceHolder, "")
var hostBinaryReplacer = strings.NewReplacer(ccBinaryTypePlaceHolder, "cc_binary_host", compatibleWithPlaceHolder, `
    target_compatible_with = select({
        "//build/bazel/platforms/os:android": ["@platforms//:incompatible"],
        "//conditions:default": [],
    }),`)

func runCcBinaryTests(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	runCcBinaryTestCase(t, tc)
	runCcHostBinaryTestCase(t, tc)
}

func runCcBinaryTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	testCase := tc
	testCase.expectedBazelTargets = append([]string{}, tc.expectedBazelTargets...)
	testCase.moduleTypeUnderTest = "cc_binary"
	testCase.moduleTypeUnderTestFactory = cc.BinaryFactory
	testCase.moduleTypeUnderTestBp2BuildMutator = cc.BinaryBp2build
	testCase.description = fmt.Sprintf("%s %s", testCase.moduleTypeUnderTest, testCase.description)
	testCase.blueprint = binaryReplacer.Replace(testCase.blueprint)
	for i, et := range testCase.expectedBazelTargets {
		testCase.expectedBazelTargets[i] = binaryReplacer.Replace(et)
	}
	t.Run(testCase.description, func(t *testing.T) {
		runBp2BuildTestCase(t, registerCcLibraryModuleTypes, testCase)
	})
}

func runCcHostBinaryTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	testCase := tc
	testCase.expectedBazelTargets = append([]string{}, tc.expectedBazelTargets...)
	testCase.moduleTypeUnderTest = "cc_binary_host"
	testCase.moduleTypeUnderTestFactory = cc.BinaryHostFactory
	testCase.moduleTypeUnderTestBp2BuildMutator = cc.BinaryHostBp2build
	testCase.description = fmt.Sprintf("%s %s", testCase.moduleTypeUnderTest, testCase.description)
	testCase.blueprint = hostBinaryReplacer.Replace(testCase.blueprint)
	for i, et := range testCase.expectedBazelTargets {
		testCase.expectedBazelTargets[i] = hostBinaryReplacer.Replace(et)
	}
	t.Run(testCase.description, func(t *testing.T) {
		runBp2BuildTestCase(t, registerCcLibraryModuleTypes, testCase)
	})
}

func TestSimpleCcBinary(t *testing.T) {
	tc := bp2buildTestCase{
		description: "simple",
		blueprint: `
{rule_name} {
    name: "foo",
    srcs: ["a.cc"],
    include_build_directory: false,
}
`,
		expectedBazelTargets: []string{`cc_binary(
    name = "foo",
    srcs = ["a.cc"],{target_compatible_with}
)`},
	}

	runCcBinaryTests(t, tc)
}

func TestCcBinaryWithSharedLdflagDisableFeature(t *testing.T) {
	tc := bp2buildTestCase{
		description: `ldflag "-shared" disables static_flag feature`,
		blueprint: `
{rule_name} {
    name: "foo",
    ldflags: ["-shared"],
    include_build_directory: false,
}
`,
		expectedBazelTargets: []string{`cc_binary(
    name = "foo",
    features = ["-static_flag"],
    linkopts = ["-shared"],{target_compatible_with}
)`},
	}

	runCcBinaryTests(t, tc)
}

func TestCcBinaryWithLinkStatic(t *testing.T) {
	tc := bp2buildTestCase{
		description: "link static",
		blueprint: `
{rule_name} {
    name: "foo",
    static_executable: true,
    include_build_directory: false,
}
`,
		expectedBazelTargets: []string{`cc_binary(
    name = "foo",
    linkshared = False,{target_compatible_with}
)`},
	}

	runCcBinaryTests(t, tc)
}
