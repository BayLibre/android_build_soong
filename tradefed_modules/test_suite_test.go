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
// There should be both "my-suite-of-modules" and "my-suite-of-modules.zip" rules.
func TestTestSuites(t *testing.T) {
	t.Parallel()
	ctx := android.GroupFixturePreparers(
		// PrepareForTestWithTestSuitesBuildComponents,
		// This adds two variants, one armv7-a-neon, one armv8-a
		android.PrepareForTestWithArchMutator,
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
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
                        test_suites: ["suite:2"],
		}

		android_test {
			name: "TestModule3",
			sdk_version: "current",
                        data: [":HelperApp", "data/testfile"],
                        test_suites: ["suite:2"],
		}

               java_test_host {
                        name: "SomeHostTest",
                        srcs: ["a.java"],
                        test_suites: ["suiteA", "general-tests",  "suiteB"],
               }

                test_module_tag {
                        name: "suite:2",
                        description: "something serious",
                }

                test_module_tag {
                        name: "suite1",
                        description: "first child",
                }

                test_module_group {
                        name: "the-group",
                        tags: ["suite1"],
                }

		test_suite {
                        name: "my-suite",
                        description: "explicit content",
                        config: "clever-config",
                        tags: ["suite:2"],
                        // modules: {                            module_names: [
                        tests: [
                                "TestModule1",
                                "TestModule2",
                        ]
                        //    ],                        },

                        // TODO(ron): see all_teams for more arch fun
                        // arch: {arm: { skip: true},
                        //        arm64: { skip: true}},
                }

                tradefed_xml_config {
                       name: "clever-config",
                       srcs: ["clever.xml"],
                       deps: ["helpful-preparer"],
                }

                java_library {
                       name: "helpful-preparer",
                       srcs: ["prep.java"],
                }
	`)

	// AssertDeepEquals(t, "compare maps", expectedTeams, actualTeams)
	//	AssertDeepEquals(t, "test matchup", expectedTests, actualTests)

	// Ensure the manifest contains the two named modules (how do I run it).
	// Ensure the manifest contains the two modules for the suite.

	expectedManifest := manifest{
		Name:   "my-suite",
		Config: "",
		Files: []string{
			// configs
			"config/clever.xml",
			"config/modules/helpful-preparer.jar",

			// directly listed
			"target/testcases/TestModule1/HelperApp.apk",
			"target/testcases/TestModule1/TestModule1.apk",
			"target/testcases/TestModule1/TestModule1.config",
			"target/testcases/TestModule1/data/testfile",

			// directly listed and tagged
			"target/testcases/TestModule2/HelperApp.apk",
			"target/testcases/TestModule2/TestModule2.apk",
			"target/testcases/TestModule2/TestModule2.config",
			"target/testcases/TestModule2/data/testfile",

			// tagged only
			"target/testcases/TestModule3/HelperApp.apk",
			"target/testcases/TestModule3/TestModule3.apk",
			"target/testcases/TestModule3/TestModule3.config",
			"target/testcases/TestModule3/data/testfile",
		},
	}

	variant := ""
	testCtx := ctx.ModuleForTests("my-suite", variant)
	// === manifest
	actualManifest := getTestSuiteManifest(t, ctx, "my-suite", &testCtx)
	fmt.Printf("WROTE manifest: %s\n", strings.Join(actualManifest.Files, "\n\t"))
	android.AssertDeepEquals(t, "", expectedManifest, actualManifest)

	// === packing file
	/*
			packingPath := testCtx.Output("packing.json")
			jsonBytes = []byte(android.ContentFromFileRuleForTests(t, ctx.TestContext, packingPath))
			actualPacking := map[string][]packagingInfo{}
			json.Unmarshal(jsonBytes, &actualPacking)
		fmt.Printf("%+v\n", actualPacking)
	*/

	android.AssertDeepEquals(t, "print errs: ", "love", "work")
}

// Read the json manifest file from the build rule output and return it as a map.
func getTestSuiteManifest(t *testing.T, ctx *android.TestResult, suiteName string, config *android.TestingModule) manifest {
	manifestPath := config.Output(fmt.Sprintf("suites/%s/%s.json", suiteName, suiteName))
	jsonBytes := []byte(android.ContentFromFileRuleForTests(t, ctx.TestContext, manifestPath))
	actualManifest := manifest{}
	json.Unmarshal(jsonBytes, &actualManifest)
	return actualManifest
}

/*
   % tree ~/aosp-main-with-phones/out/target/product/vsoc_x86_64/testcases/HelloWorldTests
/usr/local/google/home/rbraunstein/aosp-main-with-phones/out/target/product/vsoc_x86_64/testcases/HelloWorldTests
├── HelloWorldTests.config
└── x86_64
    └── HelloWorldTests.apk

tree ~/aosp-main-with-phones/out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_contentcapture
/usr/local/google/home/rbraunstein/aosp-main-with-phones/out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_contentcapture
├── data
│   └── broken_shortcut.xml -> ../../FrameworksServicesTests/data/broken_shortcut.xml
├── FrameworksServicesTests_contentcapture.config
├── MediaButtonReceiverHolderTestHelperApp.apk -> ../FrameworksServicesTests/MediaButtonReceiverHolderTestHelperApp.apk
├── SimpleServiceTestApp1.apk -> ../FrameworksServicesTests/SimpleServiceTestApp1.apk
├── SimpleServiceTestApp2.apk -> ../FrameworksServicesTests/SimpleServiceTestApp2.apk
├── SimpleServiceTestApp3.apk -> ../FrameworksServicesTests/SimpleServiceTestApp3.apk
├── SuspendTestApp.apk -> ../FrameworksServicesTests/SuspendTestApp.apk
├── test_module_config.manifest
└── x86_64
    ├── FrameworksServicesTests.apk -> ../../FrameworksServicesTests/x86_64/FrameworksServicesTests.apk
    └── UNUSED-FrameworksServicesTests.apk

% tree ~/aosp-main-with-phones/out/host/linux-x86/testcases/CtsDevicePolicyManagerTestCases_Permissions
/usr/local/google/home/rbraunstein/aosp-main-with-phones/out/host/linux-x86/testcases/CtsDevicePolicyManagerTestCases_Permissions
├── CtsCertInstallerApp.apk -> ../CtsDevicePolicyManagerTestCases/CtsCertInstallerApp.apk
├── CtsContactDirectoryProvider.apk -> ../CtsDevicePolicyManagerTestCases/CtsContactDirectoryProvider.apk
├── CtsCorpOwnedManagedProfile2.apk -> ../CtsDevicePolicyManagerTestCases/CtsCorpOwnedManagedProfile2.apk
├── CtsCorpOwnedManagedProfile.apk -> ../CtsDevicePolicyManagerTestCases/CtsCorpOwnedManagedProfile.apk
├── CtsCrossProfileEnabledApp.apk -> ../CtsDevicePolicyManagerTestCases/CtsCrossProfileEnabledApp.apk
├── CtsCrossProfileUserEnabledApp.apk -> ../CtsDevicePolicyManagerTestCases/CtsCrossProfileUserEnabledApp.apk
├── CtsDelegateApp.apk -> ../CtsDevicePolicyManagerTestCases/CtsDelegateApp.apk
├── CtsDeviceAdminApp23.apk -> ../CtsDevicePolicyManagerTestCases/CtsDeviceAdminApp23.apk


  771  2008-01-01 00:00   target/testcases/art_standalone_artd_tests/x86/art-gtest-jars-Main.jar
     1075  2008-01-01 00:00   target/testcases/art_standalone_artd_tests/x86/art-gtest-jars-Nested.jar
      771  2008-01-01 00:00   target/testcases/art_standalone_artd_tests/x86_64/art-gtest-jars-Main.jar
     1075  2008-01-01 00:00   target/testcases/art_standalone_artd_tests/x86_64/art-gtest-jars-Nested.jar

*/
