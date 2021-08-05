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

package filesystem

import (
	"os"
	"testing"

	"android/soong/android"
	"android/soong/cc"

	"github.com/google/blueprint/proptools"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

var fixture = android.GroupFixturePreparers(
	android.PrepareForIntegrationTestWithAndroid,
	cc.PrepareForIntegrationTestWithCc,
	PrepareForTestWithFilesystemBuildComponents,
)

func TestFileSystemDeps(t *testing.T) {
	result := fixture.RunTestWithBp(t, `
		android_filesystem {
			name: "myfilesystem",
		}
	`)

	// produces "myfilesystem.img"
	result.ModuleForTests("myfilesystem", "android_common").Output("myfilesystem.img")
}

func TestFileSystemFillsLinkerConfigWithStubLibs(t *testing.T) {
	result := fixture.RunTestWithBp(t, `
	  android_system_image {
			name: "myfilesystem",
			deps: [
				"libfoo",
        "libbar",
			],
			linker_config_src: "linker.config.json",
		}

		cc_library {
			name: "libfoo",
			stubs: {
				symbol_file: "libfoo.map.txt",
			},
		}

		cc_library {
			name: "libbar",
		}
	`)

	module := result.ModuleForTests("myfilesystem", "android_common")
	output := module.Output("system/etc/linker.config.pb")

	android.AssertStringDoesContain(t, "linker.config.pb should have libfoo",
		output.RuleParams.Command, "libfoo.so")
	android.AssertStringDoesNotContain(t, "linker.config.pb should not have libbar",
		output.RuleParams.Command, "libbar.so")
}

func TestFileSystemShouldInstallCoreVariantIfTargetBuildAppsIsSet(t *testing.T) {
	context := android.GroupFixturePreparers(
		fixture,
		android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
			variables.Unbundled_build_apps = proptools.BoolPtr(true)
		}),
	)
	result := context.RunTestWithBp(t, `
	  android_system_image {
			name: "myfilesystem",
			deps: [
				"libfoo",
			],
			linker_config_src: "linker.config.json",
		}

		cc_library {
			name: "libfoo",
			shared_libs: [
				"libbar",
			],
		}

		cc_library {
			name: "libbar",
			sdk_version: "9",
			stl: "none",
		}
	`)

	inputs := result.ModuleForTests("myfilesystem", "android_common").Output("deps.zip").Implicits
	android.AssertStringListContains(t, "", inputs.Strings(), "out/soong/.intermediates/libbar/android_arm64_armv8-a_shared/libbar.so")
}
