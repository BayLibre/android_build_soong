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
	"testing"

	"android/soong/android"
	"android/soong/sh"
)

func TestShTestLoadStatement(t *testing.T) {
	testCases := []struct {
		bazelTargets           BazelTargets
		expectedLoadStatements string
	}{
		{
			bazelTargets: BazelTargets{
				BazelTarget{
					name:      "sh_test_target",
					ruleClass: "sh_test",
					// Note: no bzlLoadLocation for native rules
				},
			},
			expectedLoadStatements: ``,
		},
	}

	for _, testCase := range testCases {
		actual := testCase.bazelTargets.LoadStatements()
		expected := testCase.expectedLoadStatements
		if actual != expected {
			t.Fatalf("Expected load statements to be %s, got %s", expected, actual)
		}
	}
}

func runShTestBinaryTestCase(t *testing.T, tc Bp2buildTestCase) {
	t.Helper()
	RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {}, tc)
}

func TestShTestSimple(t *testing.T) {
	runShTestBinaryTestCase(t, Bp2buildTestCase{
		Description:                "sh_test test",
		ModuleTypeUnderTest:        "sh_test",
		ModuleTypeUnderTestFactory: sh.ShTestFactory,
		Blueprint: `sh_test{
    name: "sts-rootcanal-sidebins",
    src: "empty.sh",
    test_suites: [
        "sts",
        "sts-lite",
    ],
    data_bins: [
        "android.hardware.bluetooth@1.1-service.sim",
        "android.hardware.bluetooth@1.1-impl-sim"
    ],
    data: ["android.hardware.bluetooth@1.1-service.sim.rc"],
    test_options:{tags: ["no-remote"],
	},
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("sh_test", "sts-rootcanal-sidebins", AttrNameToString{
				"srcs": `["empty.sh"]`,
				"data": `[
        "android.hardware.bluetooth@1.1-service.sim.rc",
        "android.hardware.bluetooth@1.1-service.sim",
        "android.hardware.bluetooth@1.1-impl-sim",
    ]`,
				"tags": `["no-remote"]`,
			})},
	})
}

func TestShTestBinaryDefaults(t *testing.T) {
	runShTestBinaryTestCase(t, Bp2buildTestCase{
		Description:                "sh_test test",
		ModuleTypeUnderTest:        "sh_test",
		ModuleTypeUnderTestFactory: sh.ShTestFactory,
		Blueprint: `sh_test {
    name: "foo",
    src: "foo.sh",
    bazel_module: { bp2build_available: true },
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("sh_test", "foo", AttrNameToString{
				"srcs": `["foo.sh"]`,
			})},
	})
}
