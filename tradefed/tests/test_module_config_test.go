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
	"testing"
)

const android_test_base_bp = `
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
			name: "base",
			sdk_version: "current",
                        data: [":HelperApp", "data/testfile"],
		}
`

func TestModuleConfigAndroidTest(t *testing.T) {

	bp := android_test_base_bp + `
                test_module_config {
                        name: "test_config",
                        base: "base",
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
	).RunTestWithBp(t, bp)

	// Ensure some basic rules exist.
	//buildOS := result.TestContext.Config().BuildOS.String()
	result.ModuleForTests("base", "android_common").Output("package-res.apk")
}

func TestModuleConfigOptions(t *testing.T) {

	bp := android_test_base_bp + `
                test_module_config {
                        name: "options_test",
                        base: "base",
                        exclude_filters: ["android.test.example.devcodelab.DevCodelabTest#testHelloFail"],
                        include_annotations: ["android.platform.test.annotations.LargeTest"],
                }
		`

	result := android.GroupFixturePreparers(
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureRegisterWithContext(tradefed.RegisterTestModuleConfigBuildComponents),
	).RunTestWithBp(t, bp)

	// TODO(rbraunstein): this should be relative to testcases, not intermediate.
	// Out .manifest is listed as installed.
	// tf_xml := android.ContentFromFileRuleForTests(t, ctx,
	// tf_xml_o := ctx.ModuleForTests("options_test", "android_common").Output("test_config_fixer/options_test.config").Output

	//buildOS := result.TestContext.Config().BuildOS.String()
	rule_cmd := result.ModuleForTests("options_test", "android_common").
		Rule("fix_test_config").RuleParams.Command
	android.AssertStringDoesContain(t, "Bad FixConfig rule inputs", rule_cmd,
		` --test-runner-options  '[{"Name":"exclude-filter","Key":"","Value":"android.test.example.devcodelab.DevCodelabTest#testHelloFail"},{"Name":"include-annotation","Key":"","Value":"android.platform.test.annotations.LargeTest"}]'`)

	/**
	ctx := result.TestContext
	config_module := ctx.ModuleForTests("options_test", "android_common")
	fixer_args := config_module.Rule("fix_test_config").Args["--test-runner-options"]
	// TODO(ron): I think this doesn't work as base doesn't have a AndroidTest.xml
	var tf_xml string
	for _, output := range ctx.ModuleForTests("options_test", "android_common").AllOutputs() {
		if strings.HasSuffix(output, "options_test.config") {
			fmt.Println("found tft")
			fmt.Println(output)
			if xml, err := ioutil.ReadFile(output); err == nil {
				tf_xml = string(xml)
				fmt.Println("XML:" + tf_xml)
			} else {
				fmt.Errorf("ERR: %v", err)
			}
		}
	}
	android.AssertStringDoesContain(t, fmt.Sprintf("bad xml [%+v], [%+v] [%s]", args, fixer_args, tf_xml), tf_xml, "LargerTest")
	*/

}

func TestModuleConfigJavaHostBase(t *testing.T) {
	bp := `
		java_test_host {
			name: "base"
		}

                test_module_config {
                        name: "test_config",
                        base: "base",
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
	).RunTestWithBp(t, bp)

	ctx := result.TestContext

	buildOS := ctx.Config().BuildOS.String()
	//module :=
	ctx.ModuleForTests("base", buildOS+"_common").Module()
	/* android.AssertDeepEquals(t, "Default installable value should be true.", proptools.BoolPtr(true),
	module.GetProperties("Installable"))
	*/
}
