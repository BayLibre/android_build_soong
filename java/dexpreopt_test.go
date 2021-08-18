// Copyright 2018 Google Inc. All rights reserved.
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
	"fmt"
	"runtime"
	"testing"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/dexpreopt"
)

func TestDexpreoptEnabled(t *testing.T) {
	tests := []struct {
		name    string
		bp      string
		enabled bool
	}{
		{
			name: "app",
			bp: `
				android_app {
					name: "foo",
					srcs: ["a.java"],
					sdk_version: "current",
				}`,
			enabled: true,
		},
		{
			name: "installable java library",
			bp: `
				java_library {
					name: "foo",
					installable: true,
					srcs: ["a.java"],
				}`,
			enabled: true,
		},
		{
			name: "java binary",
			bp: `
				java_binary {
					name: "foo",
					srcs: ["a.java"],
				}`,
			enabled: true,
		},
		{
			name: "app without sources",
			bp: `
				android_app {
					name: "foo",
					sdk_version: "current",
				}`,
			enabled: false,
		},
		{
			name: "app with libraries",
			bp: `
				android_app {
					name: "foo",
					static_libs: ["lib"],
					sdk_version: "current",
				}

				java_library {
					name: "lib",
					srcs: ["a.java"],
					sdk_version: "current",
				}`,
			enabled: true,
		},
		{
			name: "installable java library without sources",
			bp: `
				java_library {
					name: "foo",
					installable: true,
				}`,
			enabled: false,
		},
		{
			name: "static java library",
			bp: `
				java_library {
					name: "foo",
					srcs: ["a.java"],
				}`,
			enabled: false,
		},
		{
			name: "java test",
			bp: `
				java_test {
					name: "foo",
					srcs: ["a.java"],
				}`,
			enabled: false,
		},
		{
			name: "android test",
			bp: `
				android_test {
					name: "foo",
					srcs: ["a.java"],
				}`,
			enabled: false,
		},
		{
			name: "android test helper app",
			bp: `
				android_test_helper_app {
					name: "foo",
					srcs: ["a.java"],
				}`,
			enabled: false,
		},
		{
			name: "compile_dex",
			bp: `
				java_library {
					name: "foo",
					srcs: ["a.java"],
					compile_dex: true,
				}`,
			enabled: false,
		},
		{
			name: "dex_import",
			bp: `
				dex_import {
					name: "foo",
					jars: ["a.jar"],
				}`,
			enabled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := testJava(t, test.bp)

			dexpreopt := ctx.ModuleForTests("foo", "android_common").MaybeRule("dexpreopt")
			enabled := dexpreopt.Rule != nil

			if enabled != test.enabled {
				t.Fatalf("want dexpreopt %s, got %s", enabledString(test.enabled), enabledString(enabled))
			}
		})

	}
}

func enabledString(enabled bool) string {
	if enabled {
		return "enabled"
	} else {
		return "disabled"
	}
}

func TestDex2oatToolDeps(t *testing.T) {
	if runtime.GOOS != "linux" {
		// The host binary paths checked below are build OS dependent.
		t.Skipf("Unsupported build OS %s", runtime.GOOS)
	}

	preparers := android.GroupFixturePreparers(
		cc.PrepareForTestWithCcDefaultModules,
		PrepareForTestWithJavaDefaultModulesWithoutFakeDex2oatd,
		dexpreopt.PrepareForTestByEnablingDexpreopt)

	testDex2oatToolDep := func(sourceEnabled, prebuiltEnabled, prebuiltPreferred bool,
		expectedDex2oatPath string) {
		name := fmt.Sprintf("sourceEnabled:%t,prebuiltEnabled:%t,prebuiltPreferred:%t",
			sourceEnabled, prebuiltEnabled, prebuiltPreferred)
		t.Run(name, func(t *testing.T) {
			result := preparers.RunTestWithBp(t, fmt.Sprintf(`
					cc_binary {
						name: "dex2oatd",
						enabled: %t,
						host_supported: true,
					}
					cc_prebuilt_binary {
						name: "dex2oatd",
						enabled: %t,
						prefer: %t,
						host_supported: true,
						srcs: ["x86_64/bin/dex2oatd"],
					}
					java_library {
						name: "myjavalib",
					}
				`, sourceEnabled, prebuiltEnabled, prebuiltPreferred))
			pathContext := android.PathContextForTesting(result.Config)
			dex2oatPath := dexpreopt.GetCachedGlobalSoongConfig(pathContext).Dex2oat
			android.AssertStringEquals(t, "Testing "+name, expectedDex2oatPath, android.NormalizePathForTesting(dex2oatPath))
		})
	}

	sourceDex2oatPath := "host/linux-x86/bin/dex2oatd"
	prebuiltDex2oatPath := ".intermediates/prebuilt_dex2oatd/linux_glibc_x86_64/dex2oatd"

	testDex2oatToolDep(true, false, false, sourceDex2oatPath)
	testDex2oatToolDep(true, true, false, sourceDex2oatPath)
	testDex2oatToolDep(true, true, true, prebuiltDex2oatPath)
	testDex2oatToolDep(false, true, false, prebuiltDex2oatPath)
}

func TestUsesLibraries(t *testing.T) {
	bp := `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			api_packages: ["foo"],
			sdk_version: "current",
		}

		java_sdk_library {
			name: "qux",
			srcs: ["a.java"],
			api_packages: ["qux"],
			sdk_version: "current",
		}

		java_sdk_library {
			name: "quuz",
			srcs: ["a.java"],
			api_packages: ["quuz"],
			sdk_version: "current",
		}

		java_sdk_library {
			name: "fred",
			srcs: ["a.java"],
			api_packages: ["fred"],
			sdk_version: "current",
		}

		java_sdk_library {
			name: "bar",
			srcs: ["a.java"],
			api_packages: ["bar"],
			sdk_version: "current",
		}

		java_sdk_library {
			name: "runtime-library",
			srcs: ["a.java"],
			sdk_version: "current",
		}

		java_library {
			name: "static-runtime-helper",
			srcs: ["a.java"],
			libs: ["runtime-library"],
			sdk_version: "current",
		}

		java_library {
			name: "runtime-required-x",
			srcs: ["a.java"],
			installable: true,
			sdk_version: "current",
		}

		java_library {
			name: "runtime-optional-x",
			srcs: ["a.java"],
			installable: true,
			sdk_version: "current",
		}

		android_library {
			name: "static-x",
			uses_libs: ["runtime-required-x"],
			optional_uses_libs: ["runtime-optional-x"],
			sdk_version: "current",
		}

		java_library {
			name: "runtime-required-y",
			srcs: ["a.java"],
			installable: true,
			sdk_version: "current",
		}

		java_library {
			name: "runtime-optional-y",
			srcs: ["a.java"],
			installable: true,
			sdk_version: "current",
		}

		java_library {
			name: "static-y",
			srcs: ["a.java"],
			uses_libs: ["runtime-required-y"],
			optional_uses_libs: ["runtime-optional-y"],
			sdk_version: "current",
		}

		// A library that has to use "provides_uses_lib", because:
		//    - it is not an SDK library
		//    - its library name is different from its module name
		java_library {
			name: "non-sdk-lib",
			provides_uses_lib: "com.non.sdk.lib",
			installable: true,
			srcs: ["a.java"],
		}

		android_app {
			name: "app",
			srcs: ["a.java"],
			libs: [
				"qux",
				"quuz.stubs"
			],
			static_libs: [
				"static-runtime-helper",
				// statically linked component libraries should not pull their SDK libraries,
				// so "fred" should not be added to class loader context
				"fred.stubs",
				"static-x",
				"static-y",
			],
			uses_libs: [
				"foo",
				"non-sdk-lib"
			],
			sdk_version: "current",
			optional_uses_libs: [
				"bar",
				"baz",
			],
		}

		android_app_import {
			name: "prebuilt",
			apk: "prebuilts/apk/app.apk",
			certificate: "platform",
			uses_libs: [
				"foo",
				"non-sdk-lib",
				"android.test.runner"
			],
			optional_uses_libs: [
				"bar",
				"baz",
			],
		}
	`

	result := android.GroupFixturePreparers(
		prepareForJavaTest,
		PrepareForTestWithJavaSdkLibraryFiles,
		FixtureWithLastReleaseApis("runtime-library", "foo", "quuz", "qux", "bar", "fred"),
		android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
			variables.MissingUsesLibraries = []string{"baz"}
		}),
	).RunTestWithBp(t, bp)

	app := result.ModuleForTests("app", "android_common")
	prebuilt := result.ModuleForTests("prebuilt", "android_common")

	// Test that implicit dependencies on java_sdk_library instances are passed to the manifest.
	// This should not include explicit `uses_libs`/`optional_uses_libs` entries.
	actualManifestFixerArgs := app.Output("manifest_fixer/AndroidManifest.xml").Args["args"]
	expectManifestFixerArgs := `--extract-native-libs=true ` +
		`--uses-library qux ` +
		`--uses-library quuz ` +
		`--uses-library foo ` + // TODO(b/132357300): "foo" should not be passed to manifest_fixer
		`--uses-library com.non.sdk.lib ` + // TODO(b/132357300): "com.non.sdk.lib" should not be passed to manifest_fixer
		`--uses-library runtime-library ` +
		`--uses-library runtime-required-x ` + // TODO(b/132357300): "runtime-required-x" should not be passed to manifest_fixer
		`--uses-library runtime-required-y ` + // TODO(b/132357300): "runtime-required-y" should not be passed to manifest_fixer
		`--optional-uses-library bar ` + // TODO(b/132357300): "bar" should not be passed to manifest_fixer
		`--optional-uses-library runtime-optional-x ` + // TODO(b/132357300): "runtime-optional-x" should not be passed to manifest_fixer
		`--optional-uses-library runtime-optional-y` // TODO(b/132357300): "runtime-optional-y" should not be passed to manifest_fixer
	android.AssertStringEquals(t, "manifest_fixer args", expectManifestFixerArgs, actualManifestFixerArgs)

	// Test that all libraries are verified (library order matters).
	verifyCmd := app.Rule("verify_uses_libraries").RuleParams.Command
	verifyArgs := `--uses-library foo ` +
		`--uses-library com.non.sdk.lib ` +
		`--uses-library qux ` +
		`--uses-library quuz ` +
		`--uses-library runtime-library ` +
		`--uses-library runtime-required-x ` +
		`--uses-library runtime-required-y ` +
		`--optional-uses-library bar ` +
		`--optional-uses-library baz ` +
		`--optional-uses-library runtime-optional-x ` +
		`--optional-uses-library runtime-optional-y `
	android.AssertStringDoesContain(t, "verify cmd args", verifyCmd, verifyArgs)

	// Test that all libraries are verified for an APK (library order matters).
	verifyApkCmd := prebuilt.Rule("verify_uses_libraries").RuleParams.Command
	verifyApkArgs := `--uses-library foo ` +
		`--uses-library com.non.sdk.lib ` +
		`--uses-library android.test.runner ` +
		`--optional-uses-library bar ` +
		`--optional-uses-library baz `
	android.AssertStringDoesContain(t, "verify apk cmd args", verifyApkCmd, verifyApkArgs)

	// Test that all present libraries are preopted, including implicit SDK dependencies, possibly stubs
	cmd := app.Rule("dexpreopt").RuleParams.Command
	w := `--target-context-for-sdk any ` +
		`PCL[/system/framework/qux.jar]#` +
		`PCL[/system/framework/quuz.jar]#` +
		`PCL[/system/framework/foo.jar]#` +
		`PCL[/system/framework/non-sdk-lib.jar]#` +
		`PCL[/system/framework/bar.jar]#` +
		`PCL[/system/framework/runtime-library.jar]#` +
		`PCL[/system/framework/runtime-required-x.jar]#` +
		`PCL[/system/framework/runtime-optional-x.jar]#` +
		`PCL[/system/framework/runtime-required-y.jar]#` +
		`PCL[/system/framework/runtime-optional-y.jar] `
	android.AssertStringDoesContain(t, "dexpreopt app cmd args", cmd, w)

	// Test conditional context for target SDK version 28.
	android.AssertStringDoesContain(t, "dexpreopt app cmd 28", cmd,
		`--target-context-for-sdk 28`+
			` PCL[/system/framework/org.apache.http.legacy.jar] `)

	// Test conditional context for target SDK version 29.
	android.AssertStringDoesContain(t, "dexpreopt app cmd 29", cmd,
		`--target-context-for-sdk 29`+
			` PCL[/system/framework/android.hidl.manager-V1.0-java.jar]`+
			`#PCL[/system/framework/android.hidl.base-V1.0-java.jar] `)

	// Test conditional context for target SDK version 30.
	// "android.test.mock" is absent because "android.test.runner" is not used.
	android.AssertStringDoesContain(t, "dexpreopt app cmd 30", cmd,
		`--target-context-for-sdk 30`+
			` PCL[/system/framework/android.test.base.jar] `)

	cmd = prebuilt.Rule("dexpreopt").RuleParams.Command
	android.AssertStringDoesContain(t, "dexpreopt prebuilt cmd", cmd,
		`--target-context-for-sdk any`+
			` PCL[/system/framework/foo.jar]`+
			`#PCL[/system/framework/non-sdk-lib.jar]`+
			`#PCL[/system/framework/android.test.runner.jar]`+
			`#PCL[/system/framework/bar.jar] `)

	// Test conditional context for target SDK version 30.
	// "android.test.mock" is present because "android.test.runner" is used.
	android.AssertStringDoesContain(t, "dexpreopt prebuilt cmd 30", cmd,
		`--target-context-for-sdk 30`+
			` PCL[/system/framework/android.test.base.jar]`+
			`#PCL[/system/framework/android.test.mock.jar] `)
}

func TestDexpreoptBcp(t *testing.T) {
	bp := `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			api_packages: ["foo"],
			sdk_version: "current",
		}

		java_sdk_library {
			name: "bar",
			srcs: ["a.java"],
			api_packages: ["bar"],
			permitted_packages: ["bar"],
			sdk_version: "current",
		}

		android_app {
			name: "app",
			srcs: ["a.java"],
			sdk_version: "current",
		}
	`

	testCases := []struct {
		name   string
		with   bool
		expect string
	}{
		{
			name:   "with updatable bcp",
			with:   true,
			expect: "/system/framework/foo.jar:/system/framework/bar.jar",
		},
		{
			name:   "without updatable bcp",
			with:   false,
			expect: "/system/framework/foo.jar",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := android.GroupFixturePreparers(
				prepareForJavaTest,
				PrepareForTestWithJavaSdkLibraryFiles,
				FixtureWithLastReleaseApis("runtime-library", "foo", "bar"),
				dexpreopt.FixtureSetBootJars("platform:foo"),
				dexpreopt.FixtureSetApexBootJars("platform:bar"),
				dexpreopt.FixtureSetPreoptWithUpdatableBcp(test.with),
			).RunTestWithBp(t, bp)

			app := result.ModuleForTests("app", "android_common")
			cmd := app.Rule("dexpreopt").RuleParams.Command
			bcp := " -Xbootclasspath-locations:" + test.expect + " " // space at the end matters
			android.AssertStringDoesContain(t, "dexpreopt app bcp", cmd, bcp)
		})
	}
}
