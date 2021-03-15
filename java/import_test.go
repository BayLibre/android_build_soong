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
	"strings"
	"testing"

	"android/soong/android"
)

func TestPrebuilts(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java", ":stubs-source"],
			libs: ["bar", "sdklib"],
			static_libs: ["baz"],
		}

		java_import {
			name: "bar",
			jars: ["a.jar"],
		}

		java_import {
			name: "baz",
			jars: ["b.jar"],
			sdk_version: "current",
			compile_dex: true,
		}

		dex_import {
			name: "qux",
			jars: ["b.jar"],
		}

		java_sdk_library_import {
			name: "sdklib",
			public: {
				jars: ["c.jar"],
			},
		}

		prebuilt_stubs_sources {
			name: "stubs-source",
			srcs: ["stubs/sources"],
		}

		java_test_import {
			name: "test",
			jars: ["a.jar"],
			test_suites: ["cts"],
			test_config: "AndroidTest.xml",
		}
		`)

	fooModule := ctx.ModuleForTests("foo", "android_common")
	javac := fooModule.Rule("javac")
	combineJar := ctx.ModuleForTests("foo", "android_common").Description("for javac")
	barModule := ctx.ModuleForTests("bar", "android_common")
	barJar := barModule.Rule("combineJar").Output
	bazModule := ctx.ModuleForTests("baz", "android_common")
	bazJar := bazModule.Rule("combineJar").Output
	sdklibStubsJar := ctx.ModuleForTests("sdklib.stubs", "android_common").Rule("combineJar").Output

	fooLibrary := fooModule.Module().(*Library)
	assertDeepEquals(t, "foo java sources incorrect",
		[]string{"a.java"}, fooLibrary.compiledJavaSrcs.Strings())

	assertDeepEquals(t, "foo java source jars incorrect",
		[]string{".intermediates/stubs-source/android_common/stubs-source-stubs.srcjar"},
		android.NormalizePathsForTesting(fooLibrary.compiledSrcJars))

	if !strings.Contains(javac.Args["classpath"], barJar.String()) {
		t.Errorf("foo classpath %v does not contain %q", javac.Args["classpath"], barJar.String())
	}

	barDexJar := barModule.Module().(*Import).DexJarBuildPath()
	if barDexJar != nil {
		t.Errorf("bar dex jar build path expected to be nil, got %q", barDexJar)
	}

	if !strings.Contains(javac.Args["classpath"], sdklibStubsJar.String()) {
		t.Errorf("foo classpath %v does not contain %q", javac.Args["classpath"], sdklibStubsJar.String())
	}

	if len(combineJar.Inputs) != 2 || combineJar.Inputs[1].String() != bazJar.String() {
		t.Errorf("foo combineJar inputs %v does not contain %q", combineJar.Inputs, bazJar.String())
	}

	bazDexJar := bazModule.Module().(*Import).DexJarBuildPath().String()
	expectedDexJar := buildDir + "/.intermediates/baz/android_common/dex/baz.jar"
	if bazDexJar != expectedDexJar {
		t.Errorf("baz dex jar build path expected %q, got %q", expectedDexJar, bazDexJar)
	}

	ctx.ModuleForTests("qux", "android_common").Rule("Cp")
}

func assertDeepEquals(t *testing.T, message string, expected interface{}, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s: expected %q, found %q", message, expected, actual)
	}
}

func TestPrebuiltStubsSources(t *testing.T) {
	test := func(t *testing.T, sourcesPath string, expectedInputs []string) {
		ctx, _ := testJavaWithFS(t, fmt.Sprintf(`
prebuilt_stubs_sources {
  name: "stubs-source",
	srcs: ["%s"],
}`, sourcesPath), map[string][]byte{
			"stubs/sources/pkg/A.java": nil,
			"stubs/sources/pkg/B.java": nil,
		})

		zipSrc := ctx.ModuleForTests("stubs-source", "android_common").Rule("zip_src")
		if expected, actual := expectedInputs, zipSrc.Inputs.Strings(); !reflect.DeepEqual(expected, actual) {
			t.Errorf("mismatch of inputs to soong_zip: expected %q, actual %q", expected, actual)
		}
	}

	t.Run("empty/missing directory", func(t *testing.T) {
		test(t, "empty-directory", []string{})
	})

	t.Run("non-empty set of sources", func(t *testing.T) {
		test(t, "stubs/sources", []string{
			"stubs/sources/pkg/A.java",
			"stubs/sources/pkg/B.java",
		})
	})
}

func TestJavaImport(t *testing.T) {
	config := testConfig(nil, "", map[string][]byte{
		"libcore/Android.bp": []byte(`
				java_import {
						name: "core",
						sdk_version: "none",
				}

				filegroup {
					name: "core-jar",
					srcs: [":core{.jar}"],
				}
`),
	})
	ctx := testContext(config)
	run(t, ctx, config)
}

func TestAidlExportIncludeDirsFromImports(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["aidl/foo/IFoo.aidl"],
			libs: ["bar"],
		}

		java_import {
			name: "bar",
			jars: ["a.jar"],
			aidl: {
				export_include_dirs: ["aidl/bar"],
			},
		}
	`)

	aidlCommand := ctx.ModuleForTests("foo", "android_common").Rule("aidl").RuleParams.Command
	expectedAidlFlag := "-Iaidl/bar"
	if !strings.Contains(aidlCommand, expectedAidlFlag) {
		t.Errorf("aidl command %q does not contain %q", aidlCommand, expectedAidlFlag)
	}
}
