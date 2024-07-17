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
	`)

	// TODO(ron):
	// There should be two outputs
	//   1) The manifest of test dirs
	//      # see getTeamProtoOutput.
	//   2) The zip of files in the manifest.
	// The manifest should be always generated?
	// The zip should be generated when its a target?
	/*
		var teams *team_proto.AllTeams
		teams = getTeamProtoOutput(t, ctx)
	*/

	// map of module name -> trendy team name.
	// AssertDeepEquals(t, "compare maps", expectedTeams, actualTeams)
	//	AssertDeepEquals(t, "test matchup", expectedTests, actualTests)

	// Ensure the manifest contains the two named modules (how do I run it).
	// Ensure the manifest contains the two modules for the suite.
	expectedManifest := map[string][]string{
		"TestModule1": []string{
			"testcases/TestModule1/android_common/TestModule1.config",
			"testcases/TestModule1/android_common/HelperApp.apk",
			"testcases/TestModule1/android_common/TestModule1.apk",
			"testcases/TestModule1/android_common/data/testfile",
		}}
	actualManifest := getTestSuiteManifest(t, ctx)
	android.AssertDeepEquals(t, "", expectedManifest, actualManifest)
}

// Read the json manifest file from the build rule output and return it as a map.
func getTestSuiteManifest(t *testing.T, ctx *android.TestResult) map[string][]packagingInfo {
	config := ctx.SingletonForTests("all_test_suites")
	allOutputs := config.AllOutputs()

	// TODO(rbraunstein): fix to deal with multiple outputs
	manifestPath := allOutputs[0]

	out := config.MaybeOutput(manifestPath)
	var manifest map[string][]packagingInfo
	jsonBytes := []byte(android.ContentFromFileRuleForTests(t, ctx.TestContext, out))
	json.Unmarshal(jsonBytes, &manifest)
	return manifest
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
