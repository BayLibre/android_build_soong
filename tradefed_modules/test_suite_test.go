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
	"slices"
	"testing"
)

func TestTestSuites(t *testing.T) {
	t.Parallel()
	ctx := android.GroupFixturePreparers(
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
	).RunTestWithBp(t, `
		android_test {
			name: "TestModule1",
			sdk_version: "current",
		}

		android_test {
			name: "TestModule2",
			sdk_version: "current",
		}

		test_suite {
			name: "my-suite",
			description: "a test suite",
			tests: [
				"TestModule1",
				"TestModule2",
			]
		}
	`)
<<<<<<< PATCH SET (aa3d4c Implement test_suite.)
	manifestPath := ctx.ModuleForTests("my-suite", "").Output("out/soong/packaging/my-suite/my-suite.json")
	var actual testSuiteManifest
	if err := json.Unmarshal([]byte(android.ContentFromFileRuleForTests(t, ctx.TestContext, manifestPath)), &actual); err != nil {
		t.Errorf("failed to unmarshal manifest: %v", err)
||||||| BASE
	manifestPath := ctx.ModuleForTests("my-suite", "").Output("out/soong/packaging/my-suite/my-suite.json")
	got := android.ContentFromFileRuleForTests(t, ctx.TestContext, manifestPath)
	want := `{"name": "my-suite"}` + "\n"
	if got != want {
		t.Errorf("my-suite.json content was %q, want %q", got, want)
=======
	manifestPath := ctx.ModuleForTests("my-suite", "").Output("out/soong/test_suites/my-suite/my-suite.json")
	got := android.ContentFromFileRuleForTests(t, ctx.TestContext, manifestPath)
	want := `{"name": "my-suite"}` + "\n"
	if got != want {
		t.Errorf("my-suite.json content was %q, want %q", got, want)
>>>>>>> BASE      (aa37c1 Create out/soong/packaging directory and put a placeholder m)
	}
	slices.Sort(actual.Files)

	expected := testSuiteManifest{
		Name: "my-suite",
		Files: []string{
			"target/testcases/TestModule1/TestModule1.config",
			"target/testcases/TestModule1/arm64/TestModule1.apk",
			"target/testcases/TestModule2/TestModule2.config",
			"target/testcases/TestModule2/arm64/TestModule2.apk",
		},
	}

	android.AssertDeepEquals(t, "manifests differ", expected, actual)
}

func TestTestSuitesNotInstalledInTestcases(t *testing.T) {
	t.Parallel()
	android.GroupFixturePreparers(
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
	).ExtendWithErrorHandler(android.FixtureExpectsAllErrorsToMatchAPattern([]string{
		`"SomeHostTest" is not installed in testcases`,
	})).RunTestWithBp(t, `
			java_test_host {
				name: "SomeHostTest",
				srcs: ["a.java"],
			}
			test_suite {
				name: "my-suite",
				description: "a test suite",
				tests: [
					"SomeHostTest",
				]
			}
	`)
}
