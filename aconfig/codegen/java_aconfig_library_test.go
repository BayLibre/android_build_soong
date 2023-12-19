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

package codegen

import (
	"fmt"
	"testing"

	"android/soong/android"
	"android/soong/dexpreopt"
	"android/soong/java"
)

// Note: These tests cover the code in the java package. It'd be ideal of that code could
// be in the aconfig package.

// With the bp parameter that defines a my_module, make sure it has the LOCAL_ACONFIG_FILES entries
func runJavaAndroidMkTest(t *testing.T, bp string) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		java.PrepareForTestWithJavaDefaultModules).
		ExtendWithErrorHandler(android.FixtureExpectsNoErrors).
		RunTestWithBp(t, bp+`
			aconfig_declarations {
				name: "my_aconfig_declarations_foo",
				package: "com.example.package.foo",
				srcs: ["foo.aconfig"],
			}

			java_aconfig_library {
				name: "my_java_aconfig_library_foo",
				aconfig_declarations: "my_aconfig_declarations_foo",
			}

			aconfig_declarations {
				name: "my_aconfig_declarations_bar",
				package: "com.example.package.bar",
				srcs: ["bar.aconfig"],
			}

			java_aconfig_library {
				name: "my_java_aconfig_library_bar",
				aconfig_declarations: "my_aconfig_declarations_bar",
			}
		`)

	module := result.ModuleForTests("my_module", "android_common").Module()

	entry := android.AndroidMkEntriesForTest(t, result.TestContext, module)[0]

	makeVar := entry.EntryMap["LOCAL_ACONFIG_FILES"]
	android.AssertIntEquals(t, "len(LOCAL_ACONFIG_FILES)", 1, len(makeVar))
	android.EnsureListContainsSuffix(t, makeVar, "android_common/aconfig_merged.pb")
}

func TestAndroidMkJavaLibrary(t *testing.T) {
	bp := `
		java_library {
			name: "my_module",
			srcs: [
				"src/foo.java",
			],
			static_libs: [
				"my_java_aconfig_library_foo",
				"my_java_aconfig_library_bar",
			],
			platform_apis: true,
		}
	`

	runJavaAndroidMkTest(t, bp)
}

func TestAndroidMkAndroidApp(t *testing.T) {
	bp := `
		android_app {
			name: "my_module",
			srcs: [
				"src/foo.java",
			],
			static_libs: [
				"my_java_aconfig_library_foo",
				"my_java_aconfig_library_bar",
			],
			platform_apis: true,
		}
	`

	runJavaAndroidMkTest(t, bp)
}

func TestAndroidMkBinary(t *testing.T) {
	bp := `
		java_binary {
			name: "my_module",
			srcs: [
				"src/foo.java",
			],
			static_libs: [
				"my_java_aconfig_library_foo",
				"my_java_aconfig_library_bar",
			],
			platform_apis: true,
			main_class: "foo",
		}
	`

	runJavaAndroidMkTest(t, bp)
}

func TestAndroidMkAndroidLibrary(t *testing.T) {
	bp := `
		android_library {
			name: "my_module",
			srcs: [
				"src/foo.java",
			],
			static_libs: [
				"my_java_aconfig_library_foo",
				"my_java_aconfig_library_bar",
			],
			platform_apis: true,
		}
	`

	runJavaAndroidMkTest(t, bp)
}

func TestAndroidMkBinaryThatLinksAgainstAar(t *testing.T) {
	// Tests AndroidLibrary's propagation of flags through JavaInfo
	bp := `
		android_library {
			name: "some_library",
			srcs: [
				"src/foo.java",
			],
			static_libs: [
				"my_java_aconfig_library_foo",
				"my_java_aconfig_library_bar",
			],
			platform_apis: true,
		}
		java_binary {
			name: "my_module",
			srcs: [
				"src/bar.java",
			],
			static_libs: [
				"some_library",
			],
			platform_apis: true,
			main_class: "foo",
		}
	`

	runJavaAndroidMkTest(t, bp)
}

func testCodegenMode(t *testing.T, bpMode string, ruleMode string) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		java.PrepareForTestWithJavaDefaultModules).
		ExtendWithErrorHandler(android.FixtureExpectsNoErrors).
		RunTestWithBp(t, fmt.Sprintf(`
			aconfig_declarations {
				name: "my_aconfig_declarations",
				package: "com.example.package",
				srcs: ["foo.aconfig"],
			}

			java_aconfig_library {
				name: "my_java_aconfig_library",
				aconfig_declarations: "my_aconfig_declarations",
				%s
			}
		`, bpMode))

	module := result.ModuleForTests("my_java_aconfig_library", "android_common")
	rule := module.Rule("java_aconfig_library")
	android.AssertStringEquals(t, "rule must contain test mode", rule.Args["mode"], ruleMode)
}

func testCodegenModeWithError(t *testing.T, bpMode string, err string) {
	android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		java.PrepareForTestWithJavaDefaultModules).
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(err)).
		RunTestWithBp(t, fmt.Sprintf(`
			aconfig_declarations {
				name: "my_aconfig_declarations",
				package: "com.example.package",
				srcs: ["foo.aconfig"],
			}

			java_aconfig_library {
				name: "my_java_aconfig_library",
				aconfig_declarations: "my_aconfig_declarations",
				%s
			}
		`, bpMode))
}

func TestDefaultProdMode(t *testing.T) {
	testCodegenMode(t, "", "production")
}

func TestProdMode(t *testing.T) {
	testCodegenMode(t, "mode: `production`,", "production")
}

func TestTestMode(t *testing.T) {
	testCodegenMode(t, "mode: `test`,", "test")
}

func TestExportedMode(t *testing.T) {
	testCodegenMode(t, "mode: `exported`,", "exported")
}

func TestForceReadOnlyMode(t *testing.T) {
	testCodegenMode(t, "mode: `force-read-only`,", "force-read-only")
}

func TestUnsupportedMode(t *testing.T) {
	testCodegenModeWithError(t, "mode: `unsupported`,", "mode: \"unsupported\" is not a supported mode")
}

func TestRepackageBootclasspathfragment(t *testing.T) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		dexpreopt.PrepareForTestByEnablingDexpreopt,
		java.PrepareForTestWithJavaDefaultModules).
		ExtendWithErrorHandler(android.FixtureExpectsNoErrors).
		RunTestWithBp(t, fmt.Sprintf(`
			bootclasspath_fragment {
				name: "bcpf",
				contents: ["mylibrary1", "mylibrary2", "mylibrary4"],
				hidden_api: {
					split_packages: [],
				},
			}

			java_library {
				name: "mylibrary1",
				static_libs: ["my_java_aconfig_library_1"],
				installable: true,
			}

			java_library {
				name: "mylibrary2",
				static_libs: ["my_java_aconfig_library_1"],
				installable: true,
				jarjar_rules: "abc.txt",
			}

			java_library {
				name: "mylibrary3",
				static_libs: ["my_java_aconfig_library_1"],
				installable: true,
			}

			java_library {
				name: "mylibrary4",
				srcs: [":my_java_aconfig_library_1{.generated_srcjars}"],
				installable: true,
			}

			java_aconfig_library {
				name: "my_java_aconfig_library_1",
				aconfig_declarations: "my_aconfig_declarations_1",
			}

			aconfig_declarations {
				name: "my_aconfig_declarations_1",
				package: "com.example.packageone",
				srcs: ["foo.aconfig"],
			}
		`))

	context := result.TestContext.OtherModuleProviderAdaptor()

	libraryOne, _ := result.Module("mylibrary1", "android_common").(*java.Library)
	libraryOneProvider, _ := android.OtherModuleProvider(context, libraryOne, java.RepackageProvider)
	android.AssertDeepEquals(t, "repackage provider", libraryOneProvider.PackageToPrefix, map[string]string{"com.example.packageone": ""})
	libraryOneRepackageJarjarRules := libraryOne.GetRepackageJarjarRules()
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryOneRepackageJarjarRules, libraryOne.Name())
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryOneRepackageJarjarRules, "repackaging.txt")

	libraryTwo, _ := result.Module("mylibrary2", "android_common").(*java.Library)
	libraryTwoRepackageJarjarRules := libraryTwo.GetRepackageJarjarRules()
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryTwoRepackageJarjarRules, libraryTwo.Name())
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryTwoRepackageJarjarRules, "repackaging.txt")

	libraryThree, _ := result.Module("mylibrary3", "android_common").(*java.Library)
	android.AssertStringEquals(t, "repackageJarjarRules", libraryThree.GetRepackageJarjarRules(), "")

	libraryFour, _ := result.Module("mylibrary4", "android_common").(*java.Library)
	libraryFourRepackageJarjarRules := libraryFour.GetRepackageJarjarRules()
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryFourRepackageJarjarRules, libraryFour.Name())
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryFourRepackageJarjarRules, "repackaging.txt")
}

func TestRepackageProviderAndJarjarPrefix(t *testing.T) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		java.PrepareForTestWithJavaDefaultModules).
		ExtendWithErrorHandler(android.FixtureExpectsNoErrors).
		RunTestWithBp(t, fmt.Sprintf(`
			java_library {
				name: "mylibrary0",
				static_libs: ["mylibrary1", "mylibrary2"],
			}

			java_library {
				name: "mylibrary1",
				static_libs: ["my_java_aconfig_library_1"],
			}

			java_library {
				name: "mylibrary2",
				libs: ["mylibrary3"],
				static_libs: ["my_java_aconfig_library_2"],
				jarjar_prefix: "two",
			}

			java_library {
				name: "mylibrary3",
				static_libs: ["my_java_aconfig_library_3"],
			}

			java_aconfig_library {
				name: "my_java_aconfig_library_1",
				aconfig_declarations: "my_aconfig_declarations_1",
			}

			aconfig_declarations {
				name: "my_aconfig_declarations_1",
				package: "com.example.packageone",
				srcs: ["one.aconfig"],
			}

			java_aconfig_library {
				name: "my_java_aconfig_library_2",
				aconfig_declarations: "my_aconfig_declarations_2",
			}

			aconfig_declarations {
				name: "my_aconfig_declarations_2",
				package: "com.example.packagetwo",
				srcs: ["two.aconfig"],
			}

			java_aconfig_library {
				name: "my_java_aconfig_library_3",
				aconfig_declarations: "my_aconfig_declarations_3",
			}

			aconfig_declarations {
				name: "my_aconfig_declarations_3",
				package: "com.example.packagethree",
				srcs: ["three.aconfig"],
			}
		`))

	context := result.TestContext.OtherModuleProviderAdaptor()

	libraryZero := result.Module("mylibrary0", "android_common")
	libraryZeroProvider, _ := android.OtherModuleProvider(context, libraryZero, java.RepackageProvider)
	android.AssertDeepEquals(t, "repackage provider", libraryZeroProvider.PackageToPrefix, map[string]string{
		"com.example.packageone":   "",
		"com.example.packagetwo":   "two",
		"com.example.packagethree": "two",
	})

	libraryOne := result.Module("mylibrary1", "android_common")
	libraryOneProvider, _ := android.OtherModuleProvider(context, libraryOne, java.RepackageProvider)
	android.AssertDeepEquals(t, "repackage provider", libraryOneProvider.PackageToPrefix, map[string]string{
		"com.example.packageone": "",
	})

	libraryTwo := result.Module("mylibrary2", "android_common")
	libraryTwoProvider, _ := android.OtherModuleProvider(context, libraryTwo, java.RepackageProvider)
	android.AssertDeepEquals(t, "repackage provider", libraryTwoProvider.PackageToPrefix, map[string]string{
		"com.example.packagetwo":   "two",
		"com.example.packagethree": "two",
	})

	libraryThree := result.Module("mylibrary3", "android_common")
	libraryThreeProvider, _ := android.OtherModuleProvider(context, libraryThree, java.RepackageProvider)
	android.AssertDeepEquals(t, "repackage provider", libraryThreeProvider.PackageToPrefix, map[string]string{
		"com.example.packagethree": "",
	})
}

func TestRepackageSystemserverclasspathfragment(t *testing.T) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		java.PrepareForTestWithJavaDefaultModules).
		ExtendWithErrorHandler(android.FixtureExpectsNoErrors).
		RunTestWithBp(t, fmt.Sprintf(`
			systemserverclasspath_fragment {
				name: "sscpf",
				contents: ["mylibrary1"],
			}

			java_library {
				name: "mylibrary1",
				static_libs: ["my_java_aconfig_library"],
				installable: true,
			}

			java_library {
				name: "mylibrary2",
				static_libs: ["my_java_aconfig_library"],
				installable: true,
			}

			java_aconfig_library {
				name: "my_java_aconfig_library",
				aconfig_declarations: "my_aconfig_declarations",
			}

			aconfig_declarations {
				name: "my_aconfig_declarations",
				package: "com.example.package",
				srcs: ["foo.aconfig"],
			}
		`))

	libraryOne, _ := result.Module("mylibrary1", "android_common").(*java.Library)
	libraryOneRepackageJarjarRules := libraryOne.GetRepackageJarjarRules()
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryOneRepackageJarjarRules, libraryOne.Name())
	android.AssertStringDoesContain(t, "repackageJarjarRules", libraryOneRepackageJarjarRules, "repackaging.txt")

	libraryTwo, _ := result.Module("mylibrary2", "android_common").(*java.Library)
	android.AssertStringEquals(t, "repackageJarjarRules", libraryTwo.GetRepackageJarjarRules(), "")
}
