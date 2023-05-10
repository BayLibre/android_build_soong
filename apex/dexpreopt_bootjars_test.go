// Copyright 2019 Google Inc. All rights reserved.
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

package apex

import (
	"path/filepath"
	"sort"
	"testing"

	"android/soong/android"
	"android/soong/java"
)

func testDexpreoptBoot(t *testing.T, ruleFile string, expectedInputs, expectedOutputs []string) {
	bp := `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			api_packages: ["foo"],
		}

		java_library {
			name: "bar",
			srcs: ["b.java"],
			installable: true,
			system_ext_specific: true,
		}

		dex_import {
			name: "baz",
			jars: ["a.jar"],
		}

		platform_bootclasspath {
			name: "platform-bootclasspath",
			fragments: [
				{
					apex: "com.android.art",
					module: "art-bootclasspath-fragment",
				},
			],
		}

		java_library {
			name: "core-oj",
			srcs: ["core-oj.java"],
			installable: true,
			apex_available: [
				"com.android.art",
			],
		}

		bootclasspath_fragment {
			name: "art-bootclasspath-fragment",
			image_name: "art",
			contents: ["core-oj"],
			apex_available: [
				"com.android.art",
			],
			hidden_api: {
				split_packages: ["*"],
			},
		}

		apex {
			name: "com.android.art",
			key: "com.android.art.key",
			bootclasspath_fragments: ["art-bootclasspath-fragment"],
			updatable: false,
		}

		apex_key {
			name: "com.android.art.key",
			public_key: "com.android.art.avbpubkey",
			private_key: "com.android.art.pem",
		}
	`

	result := android.GroupFixturePreparers(
		java.PrepareForTestWithDexpreopt,
		java.PrepareForTestWithJavaSdkLibraryFiles,
		java.FixtureWithLastReleaseApis("foo"),
		java.FixtureConfigureBootJars("com.android.art:core-oj", "platform:foo", "system_ext:bar", "platform:baz"),
		PrepareForTestWithApexBuildComponents,
		prepareForTestWithArtApex,
	).RunTestWithBp(t, bp)

	platformBootclasspath := result.ModuleForTests("platform-bootclasspath", "android_common")
	rule := platformBootclasspath.Output(ruleFile)

	for i := range expectedInputs {
		expectedInputs[i] = filepath.Join("out/soong/dexpreopt_arm64", expectedInputs[i])
	}

	for i := range expectedOutputs {
		expectedOutputs[i] = filepath.Join("out/soong/dexpreopt_arm64", expectedOutputs[i])
	}

	inputs := rule.Implicits.Strings()
	sort.Strings(inputs)
	sort.Strings(expectedInputs)

	outputs := append(android.WritablePaths{rule.Output}, rule.ImplicitOutputs...).Strings()
	sort.Strings(outputs)
	sort.Strings(expectedOutputs)

	android.AssertStringPathsRelativeToTopEquals(t, "inputs", result.Config, expectedInputs, inputs)

	android.AssertStringPathsRelativeToTopEquals(t, "outputs", result.Config, expectedOutputs, outputs)
}

func TestDexpreoptBootJars(t *testing.T) {
	ruleFile := "boot.art"

	expectedInputs := []string{
		"dex_bootjars_input/core-oj.jar",
		"dex_bootjars_input/foo.jar",
		"dex_bootjars_input/bar.jar",
		"dex_bootjars_input/baz.jar",
		"dex_artjars/boot.prof",
		"dex_bootjars/boot.prof",
	}

	expectedOutputs := []string{
		"dex_bootjars/android/system/framework/arm64/boot.invocation",
		"dex_bootjars/android/system/framework/arm64/boot.art",
		"dex_bootjars/android/system/framework/arm64/boot-foo.art",
		"dex_bootjars/android/system/framework/arm64/boot-bar.art",
		"dex_bootjars/android/system/framework/arm64/boot-baz.art",
		"dex_bootjars/android/system/framework/arm64/boot.oat",
		"dex_bootjars/android/system/framework/arm64/boot-foo.oat",
		"dex_bootjars/android/system/framework/arm64/boot-bar.oat",
		"dex_bootjars/android/system/framework/arm64/boot-baz.oat",
		"dex_bootjars/android/system/framework/arm64/boot.vdex",
		"dex_bootjars/android/system/framework/arm64/boot-foo.vdex",
		"dex_bootjars/android/system/framework/arm64/boot-bar.vdex",
		"dex_bootjars/android/system/framework/arm64/boot-baz.vdex",
		"dex_bootjars_unstripped/android/system/framework/arm64/boot.oat",
		"dex_bootjars_unstripped/android/system/framework/arm64/boot-foo.oat",
		"dex_bootjars_unstripped/android/system/framework/arm64/boot-bar.oat",
		"dex_bootjars_unstripped/android/system/framework/arm64/boot-baz.oat",
	}

	testDexpreoptBoot(t, ruleFile, expectedInputs, expectedOutputs)
}

// Changes to the boot.zip structure may break the ART APK scanner.
func TestDexpreoptBootZip(t *testing.T) {
	ruleFile := "boot.zip"

	ctx := android.PathContextForTesting(android.TestArchConfig("", nil, "", nil))
	expectedInputs := []string{}
	for _, target := range ctx.Config().Targets[android.Android] {
		for _, ext := range []string{".art", ".oat", ".vdex"} {
			for _, suffix := range []string{"", "-foo", "-bar", "-baz"} {
				expectedInputs = append(expectedInputs,
					filepath.Join("dex_bootjars", target.Os.String(), "system/framework", target.Arch.ArchType.String(), "boot"+suffix+ext))
			}
		}
	}

	expectedOutputs := []string{
		"dex_bootjars/boot.zip",
	}

	testDexpreoptBoot(t, ruleFile, expectedInputs, expectedOutputs)
}
