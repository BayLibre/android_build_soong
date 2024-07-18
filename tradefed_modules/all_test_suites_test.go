// Copyright 2024 Google Inc. All rights reserved.
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
package tradefed_modules

import (
	"android/soong/android"
	"android/soong/java"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Ensure we create build rules for the TestSuites that we declare.
// There should be both "my-suite" and "my-suite.zip" rules.
func TestAllTestSuites(t *testing.T) {
	t.Parallel()
	ctx := android.GroupFixturePreparers(
		// PrepareForTestWithTestSuitesBuildComponents,
		// This adds two variants, one armv7-a-neon, one armv8-a
		android.PrepareForTestWithArchMutator,
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
		android.FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
			registerAllTestSuiteBuildComponents(ctx)
		}),
	).RunTestWithBp(t, `
		android_app {
			name: "foo",
			srcs: ["a.java"],
			sdk_version: "current",
		}

                android_test_helper_app {
                        name: "HelperApp",
                        srcs: ["helper.java"],
                }

		android_test {
			name: "TestModule1",
			sdk_version: "current",
                        data: [":HelperApp", "data/testfile"],
                        test_suites: ["suite1"],
		}

		android_test {
			name: "TestModule2",
			sdk_version: "current",
                        data: [":HelperApp", "data/testfile"],
                        test_suites: ["suite-2"],
		}

		android_test {
			name: "TestModule3",
			sdk_version: "current",
                        data: [":HelperApp", "data/testfile"],
                        test_suites: ["suite-2"],
		}

		test_suite {
                        name: "my-suite-of-modules",
                        modules: {
                            module_names: [
                                "TestModule1",
                                "TestModule2",
                            ],
                        },

                        // TODO(ron): see all_teams for more arch fun
                        // arch: {arm: { skip: true},
                        //        arm64: { skip: true}},
                }

		test_suite {
                        name: "my-suite-of-tags",
                        modules: {
                            suite_tags: [
                                "suite-2",
                            ],
                        },

                        // TODO(ron): see all_teams for more arch fun
                        // arch: {arm: { skip: true},
                        //        arm64: { skip: true}},
                }

		test_suite {
                        name: "mixed-suite",
                        modules: {
                            suite_tags: [
                                "suite-2",
                            ],
                            module_names: [
                                "TestModule1",
                            ],
                        },
                }
	`)

	// TODO(ron):
	// There should be two outputs
	//   1) The manifest of test dirs
	//      # see getTeamProtoOutput.
	//   2) The zip of files in the manifest.
	// The manifest should be always generated?
	// The zip should be generated when its a target?

	android.AssertDeepEquals(t, "",
		[]moduleName{
			"TestModule1",
			"TestModule2",
		},
		getTestSuiteManifest(t, ctx, "my-suite-of-modules"))

	android.AssertDeepEquals(t, "",
		[]moduleName{
			"TestModule2",
			"TestModule3",
		},
		getTestSuiteManifest(t, ctx, "my-suite-of-tags"))

	android.AssertDeepEquals(t, "",
		[]moduleName{
			"TestModule1",
			"TestModule2",
			"TestModule3",
		},
		getTestSuiteManifest(t, ctx, "mixed-suite"))
}

// Read the json manifest file from the build rule output and return it as a map.
func getTestSuiteManifest(t *testing.T, ctx *android.TestResult, manifestName string) []moduleName {
	config := ctx.SingletonForTests("all_test_suites")

	for _, manifestPath := range config.AllOutputs() {
		if strings.HasSuffix(manifestPath, fmt.Sprintf(manifestFilePattern, manifestName)) {
			fmt.Printf("MP: %s\n", manifestPath)

			out := config.MaybeOutput(manifestPath)
			var manifest []moduleName
			jsonBytes := []byte(android.ContentFromFileRuleForTests(t, ctx.TestContext, out))
			json.Unmarshal(jsonBytes, &manifest)
			return manifest
		}
	}
	t.Errorf("Manifest %s not found.", manifestName)
	return nil

}
