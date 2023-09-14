// Copyright 2023 Google Inc. All rights reserved.
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

package aconfig

import (
	"fmt"
	"testing"

	"android/soong/android"
	"android/soong/cc"
)

func testCodegenMode(t *testing.T, bpMode string, ruleMode string) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		cc.PrepareForTestWithCcDefaultModules).
		ExtendWithErrorHandler(android.FixtureExpectsNoErrors).
		RunTestWithBp(t, fmt.Sprintf(`
			aconfig_declarations {
				name: "my_aconfig_declarations",
				package: "com.example.package",
				srcs: ["foo.aconfig"],
			}

			cc_aconfig_library {
				name: "my_cc_aconfig_library",
				aconfig_declarations: "my_aconfig_declarations",
				%s
			}
		`, bpMode))

	module := result.ModuleForTests("my_cc_aconfig_library", "android_common")
	rule := module.Rule("cc_aconfig_library")
	android.AssertStringEquals(t, "rule must contain test mode", rule.Args["mode"], ruleMode)
}

func TestDefaultProdMode(t *testing.T) {
	testCodegenMode(t, "", "production")
}

func TestProdMode(t *testing.T) {
	testCodegenMode(t, "test: false,", "production")
}

func TestTestMode(t *testing.T) {
	testCodegenMode(t, "test: true,", "test")
}
