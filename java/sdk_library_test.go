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
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

func TestJavaSdkLibraryImport(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			libs: ["sdklib"],
			sdk_version: "current",
		}

		java_library {
			name: "foo.system",
			srcs: ["a.java"],
			libs: ["sdklib"],
			sdk_version: "system_current",
		}

		java_library {
			name: "foo.test",
			srcs: ["a.java"],
			libs: ["sdklib"],
			sdk_version: "test_current",
		}

		java_sdk_library_import {
			name: "sdklib",
			public: {
				jars: ["a.jar"],
			},
			system: {
				jars: ["b.jar"],
			},
			test: {
				jars: ["c.jar"],
				stub_srcs: ["c.java"],
			},
		}
		`)

	for _, scope := range []string{"", ".system", ".test"} {
		fooModule := ctx.ModuleForTests("foo"+scope, "android_common")
		javac := fooModule.Rule("javac")

		sdklibStubsJar := ctx.ModuleForTests("sdklib.stubs"+scope, "android_common").Rule("combineJar").Output
		if !strings.Contains(javac.Args["classpath"], sdklibStubsJar.String()) {
			t.Errorf("foo classpath %v does not contain %q", javac.Args["classpath"], sdklibStubsJar.String())
		}
	}

	CheckModuleDependencies(t, ctx, "sdklib", "android_common", []string{
		`prebuilt_sdklib.stubs`,
		`prebuilt_sdklib.stubs.source.test`,
		`prebuilt_sdklib.stubs.system`,
		`prebuilt_sdklib.stubs.test`,
	})
}

func TestJavaSdkLibraryImport_WithSource(t *testing.T) {
	ctx, _ := testJava(t, `
		java_sdk_library {
			name: "sdklib",
			srcs: ["a.java"],
			sdk_version: "none",
			system_modules: "none",
			public: {
				enabled: true,
			},
		}

		java_sdk_library_import {
			name: "sdklib",
			public: {
				jars: ["a.jar"],
			},
		}
		`)

	CheckModuleDependencies(t, ctx, "sdklib", "android_common", []string{
		`dex2oatd`,
		`prebuilt_sdklib`,
		`sdklib.impl`,
		`sdklib.stubs`,
		`sdklib.stubs.source`,
		`sdklib.xml`,
	})

	CheckModuleDependencies(t, ctx, "prebuilt_sdklib", "android_common", []string{
		`prebuilt_sdklib.stubs`,
		`sdklib.impl`,
		// This should be prebuilt_sdklib.stubs but is set to sdklib.stubs because the
		// dependency is added after prebuilts may have been renamed and so has to use
		// the renamed name.
		`sdklib.xml`,
	})
}

func TestJavaSdkLibraryImport_Preferred(t *testing.T) {
	ctx, _ := testJava(t, `
		java_sdk_library {
			name: "sdklib",
			srcs: ["a.java"],
			sdk_version: "none",
			system_modules: "none",
			public: {
				enabled: true,
			},
		}

		java_sdk_library_import {
			name: "sdklib",
			prefer: true,
			public: {
				jars: ["a.jar"],
			},
		}
		`)

	CheckModuleDependencies(t, ctx, "sdklib", "android_common", []string{
		`dex2oatd`,
		`prebuilt_sdklib`,
		`sdklib.impl`,
		`sdklib.stubs`,
		`sdklib.stubs.source`,
		`sdklib.xml`,
	})

	CheckModuleDependencies(t, ctx, "prebuilt_sdklib", "android_common", []string{
		`prebuilt_sdklib.stubs`,
		`sdklib.impl`,
		`sdklib.xml`,
	})
}

func TestJavaSdkLibraryEnforce(t *testing.T) {
	partitionToBpOption := func(partition string) string {
		switch partition {
		case "system":
			return ""
		case "vendor":
			return "soc_specific: true,"
		case "product":
			return "product_specific: true,"
		default:
			panic("Invalid partition group name: " + partition)
		}
	}

	type testConfigInfo struct {
		libraryType                string
		fromPartition              string
		toPartition                string
		enforceVendorInterface     bool
		enforceProductInterface    bool
		enforceJavaSdkLibraryCheck bool
		allowList                  []string
	}

	createPreparer := func(info testConfigInfo) android.FixturePreparer {
		bpFileTemplate := `
			java_library {
				name: "foo",
				srcs: ["foo.java"],
				libs: ["bar"],
				sdk_version: "current",
				%s
			}

			%s {
				name: "bar",
				srcs: ["bar.java"],
				sdk_version: "current",
				%s
			}
		`

		bpFile := fmt.Sprintf(bpFileTemplate,
			partitionToBpOption(info.fromPartition),
			info.libraryType,
			partitionToBpOption(info.toPartition))

		return android.GroupFixturePreparers(
			android.FixtureWithRootAndroidBp(bpFile),
			android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
				variables.EnforceProductPartitionInterface = proptools.BoolPtr(info.enforceProductInterface)
				if info.enforceVendorInterface {
					variables.DeviceVndkVersion = proptools.StringPtr("current")
				}
				variables.EnforceInterPartitionJavaSdkLibrary = proptools.BoolPtr(info.enforceJavaSdkLibraryCheck)
				variables.InterPartitionJavaLibraryAllowList = info.allowList
			}),
		)
	}

	runTest := func(t *testing.T, info testConfigInfo, expectedErrorPattern string) {
		t.Run(fmt.Sprintf("%#v", info), func(t *testing.T) {
			errorHandler := android.FixtureExpectsNoErrors
			if expectedErrorPattern != "" {
				errorHandler = android.FixtureExpectsAtLeastOneErrorMatchingPattern(expectedErrorPattern)
			}
			javaFixtureFactory.ExtendWithErrorHandler(errorHandler).RunTest(t, createPreparer(info))
		})
	}

	errorMessage := "is not allowed across the partitions"

	runTest(t, testConfigInfo{
		libraryType:                "java_library",
		fromPartition:              "product",
		toPartition:                "system",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: false,
	}, "")

	runTest(t, testConfigInfo{
		libraryType:                "java_library",
		fromPartition:              "product",
		toPartition:                "system",
		enforceVendorInterface:     true,
		enforceProductInterface:    false,
		enforceJavaSdkLibraryCheck: true,
	}, "")

	runTest(t, testConfigInfo{
		libraryType:                "java_library",
		fromPartition:              "product",
		toPartition:                "system",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: true,
	}, errorMessage)

	runTest(t, testConfigInfo{
		libraryType:                "java_library",
		fromPartition:              "vendor",
		toPartition:                "system",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: true,
	}, errorMessage)

	runTest(t, testConfigInfo{
		libraryType:                "java_library",
		fromPartition:              "vendor",
		toPartition:                "system",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: true,
		allowList:                  []string{"bar"},
	}, "")

	runTest(t, testConfigInfo{
		libraryType:                "java_library",
		fromPartition:              "vendor",
		toPartition:                "product",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: true,
	}, errorMessage)

	runTest(t, testConfigInfo{
		libraryType:                "java_sdk_library",
		fromPartition:              "product",
		toPartition:                "system",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: true,
	}, "")

	runTest(t, testConfigInfo{
		libraryType:                "java_sdk_library",
		fromPartition:              "vendor",
		toPartition:                "system",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: true,
	}, "")

	runTest(t, testConfigInfo{
		libraryType:                "java_sdk_library",
		fromPartition:              "vendor",
		toPartition:                "product",
		enforceVendorInterface:     true,
		enforceProductInterface:    true,
		enforceJavaSdkLibraryCheck: true,
	}, "")
}

func TestJavaSdkLibrary(t *testing.T) {
	ctx, _ := testJava(t, `
		droiddoc_exported_dir {
			name: "droiddoc-templates-sdk",
			path: ".",
		}
		java_sdk_library {
			name: "foo",
			srcs: ["a.java", "b.java"],
			api_packages: ["foo"],
		}
		java_sdk_library {
			name: "bar",
			srcs: ["a.java", "b.java"],
			api_packages: ["bar"],
		}
		java_library {
			name: "baz",
			srcs: ["c.java"],
			libs: ["foo", "bar.stubs"],
			sdk_version: "system_current",
		}
		java_sdk_library {
			name: "barney",
			srcs: ["c.java"],
			api_only: true,
		}
		java_sdk_library {
			name: "betty",
			srcs: ["c.java"],
			shared_library: false,
		}
		java_sdk_library_import {
		    name: "quuz",
				public: {
					jars: ["c.jar"],
				},
		}
		java_sdk_library_import {
		    name: "fred",
				public: {
					jars: ["b.jar"],
				},
		}
		java_sdk_library_import {
		    name: "wilma",
				public: {
					jars: ["b.jar"],
				},
				shared_library: false,
		}
		java_library {
		    name: "qux",
		    srcs: ["c.java"],
		    libs: ["baz", "fred", "quuz.stubs", "wilma", "barney", "betty"],
		    sdk_version: "system_current",
		}
		java_library {
			name: "baz-test",
			srcs: ["c.java"],
			libs: ["foo"],
			sdk_version: "test_current",
		}
		java_library {
			name: "baz-29",
			srcs: ["c.java"],
			libs: ["foo"],
			sdk_version: "system_29",
		}
		java_library {
			name: "baz-module-30",
			srcs: ["c.java"],
			libs: ["foo"],
			sdk_version: "module_30",
		}
		`)

	// check the existence of the internal modules
	ctx.ModuleForTests("foo", "android_common")
	ctx.ModuleForTests(apiScopePublic.stubsLibraryModuleName("foo"), "android_common")
	ctx.ModuleForTests(apiScopeSystem.stubsLibraryModuleName("foo"), "android_common")
	ctx.ModuleForTests(apiScopeTest.stubsLibraryModuleName("foo"), "android_common")
	ctx.ModuleForTests(apiScopePublic.stubsSourceModuleName("foo"), "android_common")
	ctx.ModuleForTests(apiScopeSystem.stubsSourceModuleName("foo"), "android_common")
	ctx.ModuleForTests(apiScopeTest.stubsSourceModuleName("foo"), "android_common")
	ctx.ModuleForTests("foo"+sdkXmlFileSuffix, "android_common")
	ctx.ModuleForTests("foo.api.public.28", "")
	ctx.ModuleForTests("foo.api.system.28", "")
	ctx.ModuleForTests("foo.api.test.28", "")

	bazJavac := ctx.ModuleForTests("baz", "android_common").Rule("javac")
	// tests if baz is actually linked to the stubs lib
	if !strings.Contains(bazJavac.Args["classpath"], "foo.stubs.system.jar") {
		t.Errorf("baz javac classpath %v does not contain %q", bazJavac.Args["classpath"],
			"foo.stubs.system.jar")
	}
	// ... and not to the impl lib
	if strings.Contains(bazJavac.Args["classpath"], "foo.jar") {
		t.Errorf("baz javac classpath %v should not contain %q", bazJavac.Args["classpath"],
			"foo.jar")
	}
	// test if baz is not linked to the system variant of foo
	if strings.Contains(bazJavac.Args["classpath"], "foo.stubs.jar") {
		t.Errorf("baz javac classpath %v should not contain %q", bazJavac.Args["classpath"],
			"foo.stubs.jar")
	}

	bazTestJavac := ctx.ModuleForTests("baz-test", "android_common").Rule("javac")
	// tests if baz-test is actually linked to the test stubs lib
	if !strings.Contains(bazTestJavac.Args["classpath"], "foo.stubs.test.jar") {
		t.Errorf("baz-test javac classpath %v does not contain %q", bazTestJavac.Args["classpath"],
			"foo.stubs.test.jar")
	}

	baz29Javac := ctx.ModuleForTests("baz-29", "android_common").Rule("javac")
	// tests if baz-29 is actually linked to the system 29 stubs lib
	if !strings.Contains(baz29Javac.Args["classpath"], "prebuilts/sdk/29/system/foo.jar") {
		t.Errorf("baz-29 javac classpath %v does not contain %q", baz29Javac.Args["classpath"],
			"prebuilts/sdk/29/system/foo.jar")
	}

	bazModule30Javac := ctx.ModuleForTests("baz-module-30", "android_common").Rule("javac")
	// tests if "baz-module-30" is actually linked to the module 30 stubs lib
	if !strings.Contains(bazModule30Javac.Args["classpath"], "prebuilts/sdk/30/module-lib/foo.jar") {
		t.Errorf("baz-module-30 javac classpath %v does not contain %q", bazModule30Javac.Args["classpath"],
			"prebuilts/sdk/30/module-lib/foo.jar")
	}

	// test if baz has exported SDK lib names foo and bar to qux
	qux := ctx.ModuleForTests("qux", "android_common")
	if quxLib, ok := qux.Module().(*Library); ok {
		sdkLibs := quxLib.ClassLoaderContexts().UsesLibs()
		if w := []string{"foo", "bar", "fred", "quuz"}; !reflect.DeepEqual(w, sdkLibs) {
			t.Errorf("qux should export %q but exports %q", w, sdkLibs)
		}
	}
}

func TestJavaSdkLibrary_StubOrImplOnlyLibs(t *testing.T) {
	ctx, _ := testJava(t, `
		java_sdk_library {
			name: "sdklib",
			srcs: ["a.java"],
			impl_only_libs: ["foo"],
			stub_only_libs: ["bar"],
		}
		java_library {
			name: "foo",
			srcs: ["a.java"],
			sdk_version: "current",
		}
		java_library {
			name: "bar",
			srcs: ["a.java"],
			sdk_version: "current",
		}
		`)

	for _, implName := range []string{"sdklib", "sdklib.impl"} {
		implJavacCp := ctx.ModuleForTests(implName, "android_common").Rule("javac").Args["classpath"]
		if !strings.Contains(implJavacCp, "/foo.jar") || strings.Contains(implJavacCp, "/bar.jar") {
			t.Errorf("%v javac classpath %v does not contain foo and not bar", implName, implJavacCp)
		}
	}
	stubName := apiScopePublic.stubsLibraryModuleName("sdklib")
	stubsJavacCp := ctx.ModuleForTests(stubName, "android_common").Rule("javac").Args["classpath"]
	if strings.Contains(stubsJavacCp, "/foo.jar") || !strings.Contains(stubsJavacCp, "/bar.jar") {
		t.Errorf("stubs javac classpath %v does not contain bar and not foo", stubsJavacCp)
	}
}

func TestJavaSdkLibrary_DoNotAccessImplWhenItIsNotBuilt(t *testing.T) {
	ctx, _ := testJava(t, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			api_only: true,
			public: {
				enabled: true,
			},
		}

		java_library {
			name: "bar",
			srcs: ["b.java"],
			libs: ["foo"],
		}
		`)

	// The bar library should depend on the stubs jar.
	barLibrary := ctx.ModuleForTests("bar", "android_common").Rule("javac")
	if expected, actual := `^-classpath .*:/[^:]*/turbine-combined/foo\.stubs\.jar$`, barLibrary.Args["classpath"]; !regexp.MustCompile(expected).MatchString(actual) {
		t.Errorf("expected %q, found %#q", expected, actual)
	}
}

func TestJavaSdkLibrary_UseSourcesFromAnotherSdkLibrary(t *testing.T) {
	testJava(t, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			api_packages: ["foo"],
			public: {
				enabled: true,
			},
		}

		java_library {
			name: "bar",
			srcs: ["b.java", ":foo{.public.stubs.source}"],
		}
		`)
}

func TestJavaSdkLibrary_AccessOutputFiles_MissingScope(t *testing.T) {
	testJavaError(t, `"foo" does not provide api scope system`, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			api_packages: ["foo"],
			public: {
				enabled: true,
			},
		}

		java_library {
			name: "bar",
			srcs: ["b.java", ":foo{.system.stubs.source}"],
		}
		`)
}

func TestJavaSdkLibrary_Deps(t *testing.T) {
	ctx, _ := testJava(t, `
		java_sdk_library {
			name: "sdklib",
			srcs: ["a.java"],
			sdk_version: "none",
			system_modules: "none",
			public: {
				enabled: true,
			},
		}
		`)

	CheckModuleDependencies(t, ctx, "sdklib", "android_common", []string{
		`dex2oatd`,
		`sdklib.impl`,
		`sdklib.stubs`,
		`sdklib.stubs.source`,
		`sdklib.xml`,
	})
}

func TestJavaSdkLibraryImport_AccessOutputFiles(t *testing.T) {
	testJava(t, `
		java_sdk_library_import {
			name: "foo",
			public: {
				jars: ["a.jar"],
				stub_srcs: ["a.java"],
				current_api: "api/current.txt",
				removed_api: "api/removed.txt",
			},
		}

		java_library {
			name: "bar",
			srcs: [":foo{.public.stubs.source}"],
			java_resources: [
				":foo{.public.api.txt}",
				":foo{.public.removed-api.txt}",
			],
		}
		`)
}

func TestJavaSdkLibraryImport_AccessOutputFiles_Invalid(t *testing.T) {
	bp := `
		java_sdk_library_import {
			name: "foo",
			public: {
				jars: ["a.jar"],
			},
		}
		`

	t.Run("stubs.source", func(t *testing.T) {
		testJavaError(t, `stubs.source not available for api scope public`, bp+`
		java_library {
			name: "bar",
			srcs: [":foo{.public.stubs.source}"],
			java_resources: [
				":foo{.public.api.txt}",
				":foo{.public.removed-api.txt}",
			],
		}
		`)
	})

	t.Run("api.txt", func(t *testing.T) {
		testJavaError(t, `api.txt not available for api scope public`, bp+`
		java_library {
			name: "bar",
			srcs: ["a.java"],
			java_resources: [
				":foo{.public.api.txt}",
			],
		}
		`)
	})

	t.Run("removed-api.txt", func(t *testing.T) {
		testJavaError(t, `removed-api.txt not available for api scope public`, bp+`
		java_library {
			name: "bar",
			srcs: ["a.java"],
			java_resources: [
				":foo{.public.removed-api.txt}",
			],
		}
		`)
	})
}

func TestJavaSdkLibrary_InvalidScopes(t *testing.T) {
	testJavaError(t, `module "foo": enabled api scope "system" depends on disabled scope "public"`, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java", "b.java"],
			api_packages: ["foo"],
			// Explicitly disable public to test the check that ensures the set of enabled
			// scopes is consistent.
			public: {
				enabled: false,
			},
			system: {
				enabled: true,
			},
		}
		`)
}

func TestJavaSdkLibrary_SdkVersion_ForScope(t *testing.T) {
	testJava(t, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java", "b.java"],
			api_packages: ["foo"],
			system: {
				enabled: true,
				sdk_version: "module_current",
			},
		}
		`)
}

func TestJavaSdkLibrary_ModuleLib(t *testing.T) {
	testJava(t, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java", "b.java"],
			api_packages: ["foo"],
			system: {
				enabled: true,
			},
			module_lib: {
				enabled: true,
			},
		}
		`)
}

func TestJavaSdkLibrary_SystemServer(t *testing.T) {
	testJava(t, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java", "b.java"],
			api_packages: ["foo"],
			system: {
				enabled: true,
			},
			system_server: {
				enabled: true,
			},
		}
		`)
}

func TestJavaSdkLibrary_MissingScope(t *testing.T) {
	testJavaError(t, `requires api scope module-lib from foo but it only has \[\] available`, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			public: {
				enabled: false,
			},
		}

		java_library {
			name: "baz",
			srcs: ["a.java"],
			libs: ["foo"],
			sdk_version: "module_current",
		}
		`)
}

func TestJavaSdkLibrary_FallbackScope(t *testing.T) {
	testJava(t, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			system: {
				enabled: true,
			},
		}

		java_library {
			name: "baz",
			srcs: ["a.java"],
			libs: ["foo"],
			// foo does not have module-lib scope so it should fallback to system
			sdk_version: "module_current",
		}
		`)
}

func TestJavaSdkLibrary_DefaultToStubs(t *testing.T) {
	ctx, _ := testJava(t, `
		java_sdk_library {
			name: "foo",
			srcs: ["a.java"],
			system: {
				enabled: true,
			},
			default_to_stubs: true,
		}

		java_library {
			name: "baz",
			srcs: ["a.java"],
			libs: ["foo"],
			// does not have sdk_version set, should fallback to module,
			// which will then fallback to system because the module scope
			// is not enabled.
		}
		`)
	// The baz library should depend on the system stubs jar.
	bazLibrary := ctx.ModuleForTests("baz", "android_common").Rule("javac")
	if expected, actual := `^-classpath .*:/[^:]*/turbine-combined/foo\.stubs.system\.jar$`, bazLibrary.Args["classpath"]; !regexp.MustCompile(expected).MatchString(actual) {
		t.Errorf("expected %q, found %#q", expected, actual)
	}
}
