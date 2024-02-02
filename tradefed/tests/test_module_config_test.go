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
package tests

import (
	"android/soong/android"
	"android/soong/java"
	"android/soong/tradefed"
	"strings"
	"testing"
)

func TestModuleConfigAndroidTest(t *testing.T) {
	bp := `
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
			name: "bar",
		        //  instrumentation_for: "foo",
			sdk_version: "current",
                        data: [":HelperApp", "data/testfile"],
		}

                test_module_config {
                        name: "test_config",
                        base: "bar",
			options: [
			    {
				name: "exclude-filter",
				value: "android.test.example.devcodelab.DevCodelabTest#testHelloFail",
			    },
			],
                }
		`

	result := android.GroupFixturePreparers(
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureRegisterWithContext(tradefed.RegisterTestModuleConfigBuildComponents),
		android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
			variables.ManifestPackageNameOverrides = []string{"foo:org.dandroid.bp"}
		}),
	).RunTestWithBp(t, bp)

	bar := result.ModuleForTests("bar", "android_common")
	res := bar.Output("package-res.apk")
	aapt2Flags := res.Args["flags"]
	e := "--rlename-instrumentation-target-package org.dandroid.bp"
	if !strings.Contains(aapt2Flags, e) {
		t.Errorf("target package renaming flag, %q is missing in aapt2 link flags, %q", e, aapt2Flags)
	}
}
