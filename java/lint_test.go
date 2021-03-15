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

package java

import (
	"strings"
	"testing"
)

func TestJavaLint(t *testing.T) {
	ctx, _ := testJavaWithFS(t, `
		java_library {
			name: "foo",
			srcs: [
				"a.java",
				"b.java",
				"c.java",
			],
			min_sdk_version: "29",
			sdk_version: "system_current",
		}
       `, map[string][]byte{
		"lint-baseline.xml": nil,
	})

	foo := ctx.ModuleForTests("foo", "android_common")
	rule := foo.Rule("lint")

	if !strings.Contains(rule.RuleParams.Command, "--baseline lint-baseline.xml") {
		t.Error("did not pass --baseline flag")
	}
}

func TestJavaLintWithoutBaseline(t *testing.T) {
	ctx, _ := testJavaWithFS(t, `
		java_library {
			name: "foo",
			srcs: [
				"a.java",
				"b.java",
				"c.java",
			],
			min_sdk_version: "29",
			sdk_version: "system_current",
		}
       `, map[string][]byte{})

	foo := ctx.ModuleForTests("foo", "android_common")
	rule := foo.Rule("lint")

	if strings.Contains(rule.RuleParams.Command, "--baseline") {
		t.Error("passed --baseline flag for non existent file")
	}
}

func TestJavaLintRequiresCustomLintFileToExist(t *testing.T) {
	config := testConfig(
		nil,
		`
		java_library {
			name: "foo",
			srcs: [
			],
			min_sdk_version: "29",
			sdk_version: "system_current",
			lint: {
				baseline_filename: "mybaseline.xml",
			},
		}
     `, map[string][]byte{
			"build/soong/java/lint_defaults.txt":                   nil,
			"prebuilts/cmdline-tools/tools/bin/lint":               nil,
			"prebuilts/cmdline-tools/tools/lib/lint-classpath.jar": nil,
			"framework/aidl":                     nil,
			"a.java":                             nil,
			"AndroidManifest.xml":                nil,
			"build/make/target/product/security": nil,
		})
	config.TestAllowNonExistentPaths = false
	testJavaErrorWithConfig(t,
		"source path \"mybaseline.xml\" does not exist",
		config,
	)
}

func TestJavaLintUsesCorrectBpConfig(t *testing.T) {
	ctx, _ := testJavaWithFS(t, `
		java_library {
			name: "foo",
			srcs: [
				"a.java",
				"b.java",
				"c.java",
			],
			min_sdk_version: "29",
			sdk_version: "system_current",
			lint: {
				error_checks: ["SomeCheck"],
				baseline_filename: "mybaseline.xml",
			},
		}
       `, map[string][]byte{
		"mybaseline.xml": nil,
	})

	foo := ctx.ModuleForTests("foo", "android_common")
	rule := foo.Rule("lint")

	if !strings.Contains(rule.RuleParams.Command, "--baseline mybaseline.xml") {
		t.Error("did not use the correct file for baseline")
	}
}
