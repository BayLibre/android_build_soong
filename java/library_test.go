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
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

func TestJavaLinkType(t *testing.T) {
	testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			libs: ["bar"],
			static_libs: ["baz"],
		}

		java_library {
			name: "bar",
			sdk_version: "current",
			srcs: ["b.java"],
		}

		java_library {
			name: "baz",
			sdk_version: "system_current",
			srcs: ["c.java"],
		}
	`)

	testJavaError(t, "consider adjusting sdk_version: OR platform_apis:", `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			libs: ["bar"],
			sdk_version: "current",
			static_libs: ["baz"],
		}

		java_library {
			name: "bar",
			sdk_version: "current",
			srcs: ["b.java"],
		}

		java_library {
			name: "baz",
			sdk_version: "system_current",
			srcs: ["c.java"],
		}
	`)

	testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			libs: ["bar"],
			sdk_version: "system_current",
			static_libs: ["baz"],
		}

		java_library {
			name: "bar",
			sdk_version: "current",
			srcs: ["b.java"],
		}

		java_library {
			name: "baz",
			sdk_version: "system_current",
			srcs: ["c.java"],
		}
	`)

	testJavaError(t, "consider adjusting sdk_version: OR platform_apis:", `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			libs: ["bar"],
			sdk_version: "system_current",
			static_libs: ["baz"],
		}

		java_library {
			name: "bar",
			sdk_version: "current",
			srcs: ["b.java"],
		}

		java_library {
			name: "baz",
			srcs: ["c.java"],
		}
	`)
}

func TestSimple(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			libs: ["bar"],
			static_libs: ["baz"],
		}

		java_library {
			name: "bar",
			srcs: ["b.java"],
		}

		java_library {
			name: "baz",
			srcs: ["c.java"],
		}
	`)

	javac := ctx.ModuleForTests("foo", "android_common").Rule("javac")
	combineJar := ctx.ModuleForTests("foo", "android_common").Description("for javac")

	if len(javac.Inputs) != 1 || javac.Inputs[0].String() != "a.java" {
		t.Errorf(`foo inputs %v != ["a.java"]`, javac.Inputs)
	}

	baz := ctx.ModuleForTests("baz", "android_common").Rule("javac").Output.String()
	barTurbine := filepath.Join(buildDir, ".intermediates", "bar", "android_common", "turbine-combined", "bar.jar")
	bazTurbine := filepath.Join(buildDir, ".intermediates", "baz", "android_common", "turbine-combined", "baz.jar")

	if !strings.Contains(javac.Args["classpath"], barTurbine) {
		t.Errorf("foo classpath %v does not contain %q", javac.Args["classpath"], barTurbine)
	}

	if !strings.Contains(javac.Args["classpath"], bazTurbine) {
		t.Errorf("foo classpath %v does not contain %q", javac.Args["classpath"], bazTurbine)
	}

	if len(combineJar.Inputs) != 2 || combineJar.Inputs[1].String() != baz {
		t.Errorf("foo combineJar inputs %v does not contain %q", combineJar.Inputs, baz)
	}
}

func TestExportedPlugins(t *testing.T) {
	type Result struct {
		library        string
		processors     string
		disableTurbine bool
	}
	var tests = []struct {
		name    string
		extra   string
		results []Result
	}{
		{
			name:    "Exported plugin is not a direct plugin",
			extra:   `java_library { name: "exports", srcs: ["a.java"], exported_plugins: ["plugin"] }`,
			results: []Result{{library: "exports", processors: "-proc:none"}},
		},
		{
			name: "Exports plugin to dependee",
			extra: `
				java_library{name: "exports", exported_plugins: ["plugin"]}
				java_library{name: "foo", srcs: ["a.java"], libs: ["exports"]}
				java_library{name: "bar", srcs: ["a.java"], static_libs: ["exports"]}
			`,
			results: []Result{
				{library: "foo", processors: "-processor com.android.TestPlugin"},
				{library: "bar", processors: "-processor com.android.TestPlugin"},
			},
		},
		{
			name: "Exports plugin to android_library",
			extra: `
				java_library{name: "exports", exported_plugins: ["plugin"]}
				android_library{name: "foo", srcs: ["a.java"],  libs: ["exports"]}
				android_library{name: "bar", srcs: ["a.java"], static_libs: ["exports"]}
			`,
			results: []Result{
				{library: "foo", processors: "-processor com.android.TestPlugin"},
				{library: "bar", processors: "-processor com.android.TestPlugin"},
			},
		},
		{
			name: "Exports plugin is not propagated via transitive deps",
			extra: `
				java_library{name: "exports", exported_plugins: ["plugin"]}
				java_library{name: "foo", srcs: ["a.java"], libs: ["exports"]}
				java_library{name: "bar", srcs: ["a.java"], static_libs: ["foo"]}
			`,
			results: []Result{
				{library: "foo", processors: "-processor com.android.TestPlugin"},
				{library: "bar", processors: "-proc:none"},
			},
		},
		{
			name: "Exports plugin appends to plugins",
			extra: `
                java_plugin{name: "plugin2", processor_class: "com.android.TestPlugin2"}
				java_library{name: "exports", exported_plugins: ["plugin"]}
				java_library{name: "foo", srcs: ["a.java"], libs: ["exports"], plugins: ["plugin2"]}
			`,
			results: []Result{
				{library: "foo", processors: "-processor com.android.TestPlugin,com.android.TestPlugin2"},
			},
		},
		{
			name: "Exports plugin to with generates_api to dependee",
			extra: `
				java_library{name: "exports", exported_plugins: ["plugin_generates_api"]}
				java_library{name: "foo", srcs: ["a.java"], libs: ["exports"]}
				java_library{name: "bar", srcs: ["a.java"], static_libs: ["exports"]}
			`,
			results: []Result{
				{library: "foo", processors: "-processor com.android.TestPlugin", disableTurbine: true},
				{library: "bar", processors: "-processor com.android.TestPlugin", disableTurbine: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := testJava(t, `
				java_plugin {
					name: "plugin",
					processor_class: "com.android.TestPlugin",
				}
				java_plugin {
					name: "plugin_generates_api",
					generates_api: true,
					processor_class: "com.android.TestPlugin",
				}
			`+test.extra)

			for _, want := range test.results {
				javac := ctx.ModuleForTests(want.library, "android_common").Rule("javac")
				if javac.Args["processor"] != want.processors {
					t.Errorf("For library %v, expected %v, found %v", want.library, want.processors, javac.Args["processor"])
				}
				turbine := ctx.ModuleForTests(want.library, "android_common").MaybeRule("turbine")
				disableTurbine := turbine.BuildParams.Rule == nil
				if disableTurbine != want.disableTurbine {
					t.Errorf("For library %v, expected disableTurbine %v, found %v", want.library, want.disableTurbine, disableTurbine)
				}
			}
		})
	}
}

func TestSdkVersionByPartition(t *testing.T) {
	testJavaError(t, "sdk_version must have a value when the module is located at vendor or product", `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			vendor: true,
		}
	`)

	testJava(t, `
		java_library {
			name: "bar",
			srcs: ["b.java"],
		}
	`)

	for _, enforce := range []bool{true, false} {
		bp := `
			java_library {
				name: "foo",
				srcs: ["a.java"],
				product_specific: true,
			}
		`

		config := testConfig(nil, bp, nil)
		config.TestProductVariables.EnforceProductPartitionInterface = proptools.BoolPtr(enforce)
		if enforce {
			testJavaErrorWithConfig(t, "sdk_version must have a value when the module is located at vendor or product", config)
		} else {
			testJavaWithConfig(t, config)
		}
	}
}

func TestArchSpecific(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			target: {
				android: {
					srcs: ["b.java"],
				},
			},
		}
	`)

	javac := ctx.ModuleForTests("foo", "android_common").Rule("javac")
	if len(javac.Inputs) != 2 || javac.Inputs[0].String() != "a.java" || javac.Inputs[1].String() != "b.java" {
		t.Errorf(`foo inputs %v != ["a.java", "b.java"]`, javac.Inputs)
	}
}

func TestResources(t *testing.T) {
	var table = []struct {
		name  string
		prop  string
		extra string
		args  string
	}{
		{
			// Test that a module with java_resource_dirs includes the files
			name: "resource dirs",
			prop: `java_resource_dirs: ["java-res"]`,
			args: "-C java-res -f java-res/a/a -f java-res/b/b",
		},
		{
			// Test that a module with java_resources includes the files
			name: "resource files",
			prop: `java_resources: ["java-res/a/a", "java-res/b/b"]`,
			args: "-C . -f java-res/a/a -f java-res/b/b",
		},
		{
			// Test that a module with a filegroup in java_resources includes the files with the
			// path prefix
			name: "resource filegroup",
			prop: `java_resources: [":foo-res"]`,
			extra: `
				filegroup {
					name: "foo-res",
					path: "java-res",
					srcs: ["java-res/a/a", "java-res/b/b"],
				}`,
			args: "-C java-res -f java-res/a/a -f java-res/b/b",
		},
		{
			// Test that a module with wildcards in java_resource_dirs has the correct path prefixes
			name: "wildcard dirs",
			prop: `java_resource_dirs: ["java-res/*"]`,
			args: "-C java-res/a -f java-res/a/a -C java-res/b -f java-res/b/b",
		},
		{
			// Test that a module exclude_java_resource_dirs excludes the files
			name: "wildcard dirs",
			prop: `java_resource_dirs: ["java-res/*"], exclude_java_resource_dirs: ["java-res/b"]`,
			args: "-C java-res/a -f java-res/a/a",
		},
		{
			// Test wildcards in java_resources
			name: "wildcard files",
			prop: `java_resources: ["java-res/**/*"]`,
			args: "-C . -f java-res/a/a -f java-res/b/b",
		},
		{
			// Test exclude_java_resources with java_resources
			name: "wildcard files with exclude",
			prop: `java_resources: ["java-res/**/*"], exclude_java_resources: ["java-res/b/*"]`,
			args: "-C . -f java-res/a/a",
		},
		{
			// Test exclude_java_resources with java_resource_dirs
			name: "resource dirs with exclude files",
			prop: `java_resource_dirs: ["java-res"], exclude_java_resources: ["java-res/b/b"]`,
			args: "-C java-res -f java-res/a/a",
		},
		{
			// Test exclude_java_resource_dirs with java_resource_dirs
			name: "resource dirs with exclude files",
			prop: `java_resource_dirs: ["java-res", "java-res2"], exclude_java_resource_dirs: ["java-res2"]`,
			args: "-C java-res -f java-res/a/a -f java-res/b/b",
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := testJavaWithFS(t, `
				java_library {
					name: "foo",
					srcs: [
						"a.java",
						"b.java",
						"c.java",
					],
					`+test.prop+`,
				}
			`+test.extra,
				map[string][]byte{
					"java-res/a/a": nil,
					"java-res/b/b": nil,
					"java-res2/a":  nil,
				},
			)

			foo := ctx.ModuleForTests("foo", "android_common").Output("withres/foo.jar")
			fooRes := ctx.ModuleForTests("foo", "android_common").Output("res/foo.jar")

			if !inList(fooRes.Output.String(), foo.Inputs.Strings()) {
				t.Errorf("foo combined jars %v does not contain %q",
					foo.Inputs.Strings(), fooRes.Output.String())
			}

			if fooRes.Args["jarArgs"] != test.args {
				t.Errorf("foo resource jar args %q is not %q",
					fooRes.Args["jarArgs"], test.args)
			}
		})
	}
}

func TestIncludeSrcs(t *testing.T) {
	ctx, _ := testJavaWithFS(t, `
		java_library {
			name: "foo",
			srcs: [
				"a.java",
				"b.java",
				"c.java",
			],
			include_srcs: true,
		}

		java_library {
			name: "bar",
			srcs: [
				"a.java",
				"b.java",
				"c.java",
			],
			java_resource_dirs: ["java-res"],
			include_srcs: true,
		}
	`, map[string][]byte{
		"java-res/a/a": nil,
		"java-res/b/b": nil,
		"java-res2/a":  nil,
	})

	// Test a library with include_srcs: true
	foo := ctx.ModuleForTests("foo", "android_common").Output("withres/foo.jar")
	fooSrcJar := ctx.ModuleForTests("foo", "android_common").Output("foo.srcjar")

	if g, w := fooSrcJar.Output.String(), foo.Inputs.Strings(); !inList(g, w) {
		t.Errorf("foo combined jars %v does not contain %q", w, g)
	}

	if g, w := fooSrcJar.Args["jarArgs"], "-C . -f a.java -f b.java -f c.java"; g != w {
		t.Errorf("foo source jar args %q is not %q", w, g)
	}

	// Test a library with include_srcs: true and resources
	bar := ctx.ModuleForTests("bar", "android_common").Output("withres/bar.jar")
	barResCombined := ctx.ModuleForTests("bar", "android_common").Output("res-combined/bar.jar")
	barRes := ctx.ModuleForTests("bar", "android_common").Output("res/bar.jar")
	barSrcJar := ctx.ModuleForTests("bar", "android_common").Output("bar.srcjar")

	if g, w := barSrcJar.Output.String(), barResCombined.Inputs.Strings(); !inList(g, w) {
		t.Errorf("bar combined resource jars %v does not contain %q", w, g)
	}

	if g, w := barRes.Output.String(), barResCombined.Inputs.Strings(); !inList(g, w) {
		t.Errorf("bar combined resource jars %v does not contain %q", w, g)
	}

	if g, w := barResCombined.Output.String(), bar.Inputs.Strings(); !inList(g, w) {
		t.Errorf("bar combined jars %v does not contain %q", w, g)
	}

	if g, w := barSrcJar.Args["jarArgs"], "-C . -f a.java -f b.java -f c.java"; g != w {
		t.Errorf("bar source jar args %q is not %q", w, g)
	}

	if g, w := barRes.Args["jarArgs"], "-C java-res -f java-res/a/a -f java-res/b/b"; g != w {
		t.Errorf("bar resource jar args %q is not %q", w, g)
	}
}

func TestGeneratedSources(t *testing.T) {
	ctx, _ := testJavaWithFS(t, `
		java_library {
			name: "foo",
			srcs: [
				"a*.java",
				":gen",
				"b*.java",
			],
		}

		genrule {
			name: "gen",
			tool_files: ["java-res/a"],
			out: ["gen.java"],
		}
	`, map[string][]byte{
		"a.java": nil,
		"b.java": nil,
	})

	javac := ctx.ModuleForTests("foo", "android_common").Rule("javac")
	genrule := ctx.ModuleForTests("gen", "").Rule("generator")

	if filepath.Base(genrule.Output.String()) != "gen.java" {
		t.Fatalf(`gen output file %v is not ".../gen.java"`, genrule.Output.String())
	}

	if len(javac.Inputs) != 3 ||
		javac.Inputs[0].String() != "a.java" ||
		javac.Inputs[1].String() != genrule.Output.String() ||
		javac.Inputs[2].String() != "b.java" {
		t.Errorf(`foo inputs %v != ["a.java", ".../gen.java", "b.java"]`, javac.Inputs)
	}
}

func TestTurbine(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			sdk_version: "14",
		}

		java_library {
			name: "bar",
			srcs: ["b.java"],
			static_libs: ["foo"],
			sdk_version: "14",
		}

		java_library {
			name: "baz",
			srcs: ["c.java"],
			libs: ["bar"],
			sdk_version: "14",
		}
		`)

	fooTurbine := ctx.ModuleForTests("foo", "android_common").Rule("turbine")
	barTurbine := ctx.ModuleForTests("bar", "android_common").Rule("turbine")
	barJavac := ctx.ModuleForTests("bar", "android_common").Rule("javac")
	barTurbineCombined := ctx.ModuleForTests("bar", "android_common").Description("for turbine")
	bazJavac := ctx.ModuleForTests("baz", "android_common").Rule("javac")

	if len(fooTurbine.Inputs) != 1 || fooTurbine.Inputs[0].String() != "a.java" {
		t.Errorf(`foo inputs %v != ["a.java"]`, fooTurbine.Inputs)
	}

	fooHeaderJar := filepath.Join(buildDir, ".intermediates", "foo", "android_common", "turbine-combined", "foo.jar")
	if !strings.Contains(barTurbine.Args["classpath"], fooHeaderJar) {
		t.Errorf("bar turbine classpath %v does not contain %q", barTurbine.Args["classpath"], fooHeaderJar)
	}
	if !strings.Contains(barJavac.Args["classpath"], fooHeaderJar) {
		t.Errorf("bar javac classpath %v does not contain %q", barJavac.Args["classpath"], fooHeaderJar)
	}
	if len(barTurbineCombined.Inputs) != 2 || barTurbineCombined.Inputs[1].String() != fooHeaderJar {
		t.Errorf("bar turbine combineJar inputs %v does not contain %q", barTurbineCombined.Inputs, fooHeaderJar)
	}
	if !strings.Contains(bazJavac.Args["classpath"], "prebuilts/sdk/14/public/android.jar") {
		t.Errorf("baz javac classpath %v does not contain %q", bazJavac.Args["classpath"],
			"prebuilts/sdk/14/public/android.jar")
	}
}

func TestSharding(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "bar",
			srcs: ["a.java","b.java","c.java"],
			javac_shard_size: 1
		}
		`)

	barHeaderJar := filepath.Join(buildDir, ".intermediates", "bar", "android_common", "turbine-combined", "bar.jar")
	for i := 0; i < 3; i++ {
		barJavac := ctx.ModuleForTests("bar", "android_common").Description("javac" + strconv.Itoa(i))
		if !strings.Contains(barJavac.Args["classpath"], barHeaderJar) {
			t.Errorf("bar javac classpath %v does not contain %q", barJavac.Args["classpath"], barHeaderJar)
		}
	}
}

func TestExcludeFileGroupInSrcs(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java", ":foo-srcs"],
			exclude_srcs: ["a.java", ":foo-excludes"],
		}

		filegroup {
			name: "foo-srcs",
			srcs: ["java-fg/a.java", "java-fg/b.java", "java-fg/c.java"],
		}

		filegroup {
			name: "foo-excludes",
			srcs: ["java-fg/a.java", "java-fg/b.java"],
		}
	`)

	javac := ctx.ModuleForTests("foo", "android_common").Rule("javac")

	if len(javac.Inputs) != 1 || javac.Inputs[0].String() != "java-fg/c.java" {
		t.Errorf(`foo inputs %v != ["java-fg/c.java"]`, javac.Inputs)
	}
}

func TestJavaLibrary(t *testing.T) {
	config := testConfig(nil, "", map[string][]byte{
		"libcore/Android.bp": []byte(`
				java_library {
						name: "core",
						sdk_version: "none",
						system_modules: "none",
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

var compilerFlagsTestCases = []struct {
	in  string
	out bool
}{
	{
		in:  "a",
		out: false,
	},
	{
		in:  "-a",
		out: true,
	},
	{
		in:  "-no-jdk",
		out: false,
	},
	{
		in:  "-no-stdlib",
		out: false,
	},
	{
		in:  "-kotlin-home",
		out: false,
	},
	{
		in:  "-kotlin-home /some/path",
		out: false,
	},
	{
		in:  "-include-runtime",
		out: false,
	},
	{
		in:  "-Xintellij-plugin-root",
		out: false,
	},
}

type mockContext struct {
	android.ModuleContext
	result bool
}

func (ctx *mockContext) PropertyErrorf(property, format string, args ...interface{}) {
	// CheckBadCompilerFlags calls this function when the flag should be rejected
	ctx.result = false
}

func TestCompilerFlags(t *testing.T) {
	for _, testCase := range compilerFlagsTestCases {
		ctx := &mockContext{result: true}
		checkKotlincFlags(ctx, []string{testCase.in})
		if ctx.result != testCase.out {
			t.Errorf("incorrect output:")
			t.Errorf("     input: %#v", testCase.in)
			t.Errorf("  expected: %#v", testCase.out)
			t.Errorf("       got: %#v", ctx.result)
		}
	}
}

// TODO(jungjw): Consider making this more robust by ignoring path order.
func checkPatchModuleFlag(t *testing.T, ctx *android.TestContext, moduleName string, expected string) {
	variables := ctx.ModuleForTests(moduleName, "android_common").Module().VariablesForTests()
	flags := strings.Split(variables["javacFlags"], " ")
	got := ""
	for _, flag := range flags {
		keyEnd := strings.Index(flag, "=")
		if keyEnd > -1 && flag[:keyEnd] == "--patch-module" {
			got = flag[keyEnd+1:]
			break
		}
	}
	if expected != got {
		t.Errorf("Unexpected patch-module flag for module %q - expected %q, but got %q", moduleName, expected, got)
	}
}

func TestPatchModule(t *testing.T) {
	t.Run("Java language level 8", func(t *testing.T) {
		// Test with legacy javac -source 1.8 -target 1.8
		bp := `
			java_library {
				name: "foo",
				srcs: ["a.java"],
				java_version: "1.8",
			}

			java_library {
				name: "bar",
				srcs: ["b.java"],
				sdk_version: "none",
				system_modules: "none",
				patch_module: "java.base",
				java_version: "1.8",
			}

			java_library {
				name: "baz",
				srcs: ["c.java"],
				patch_module: "java.base",
				java_version: "1.8",
			}
		`
		ctx, _ := testJava(t, bp)

		checkPatchModuleFlag(t, ctx, "foo", "")
		checkPatchModuleFlag(t, ctx, "bar", "")
		checkPatchModuleFlag(t, ctx, "baz", "")
	})

	t.Run("Java language level 9", func(t *testing.T) {
		// Test with default javac -source 9 -target 9
		bp := `
			java_library {
				name: "foo",
				srcs: ["a.java"],
			}

			java_library {
				name: "bar",
				srcs: ["b.java"],
				sdk_version: "none",
				system_modules: "none",
				patch_module: "java.base",
			}

			java_library {
				name: "baz",
				srcs: [
					"c.java",
					// Tests for b/150878007
					"dir/d.java",
					"dir2/e.java",
					"dir2/f.java",
					"nested/dir/g.java"
				],
				patch_module: "java.base",
			}
		`
		ctx, _ := testJava(t, bp)

		checkPatchModuleFlag(t, ctx, "foo", "")
		expected := "java.base=.:" + buildDir
		checkPatchModuleFlag(t, ctx, "bar", expected)
		expected = "java.base=" + strings.Join([]string{
			".", buildDir, "dir", "dir2", "nested", defaultModuleToPath("ext"), defaultModuleToPath("framework")}, ":")
		checkPatchModuleFlag(t, ctx, "baz", expected)
	})
}

func TestJavaLibraryWithSystemModules(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
		    name: "lib-with-source-system-modules",
		    srcs: [
		        "a.java",
		    ],
				sdk_version: "none",
				system_modules: "source-system-modules",
		}

		java_library {
				name: "source-jar",
		    srcs: [
		        "a.java",
		    ],
		}

		java_system_modules {
				name: "source-system-modules",
				libs: ["source-jar"],
		}

		java_library {
		    name: "lib-with-prebuilt-system-modules",
		    srcs: [
		        "a.java",
		    ],
				sdk_version: "none",
				system_modules: "prebuilt-system-modules",
		}

		java_import {
				name: "prebuilt-jar",
				jars: ["a.jar"],
		}

		java_system_modules_import {
				name: "prebuilt-system-modules",
				libs: ["prebuilt-jar"],
		}
		`)

	checkBootClasspathForSystemModule(t, ctx, "lib-with-source-system-modules", "/source-jar.jar")

	checkBootClasspathForSystemModule(t, ctx, "lib-with-prebuilt-system-modules", "/prebuilt-jar.jar")
}

func checkBootClasspathForSystemModule(t *testing.T, ctx *android.TestContext, moduleName string, expectedSuffix string) {
	javacRule := ctx.ModuleForTests(moduleName, "android_common").Rule("javac")
	bootClasspath := javacRule.Args["bootClasspath"]
	if strings.HasPrefix(bootClasspath, "--system ") && strings.HasSuffix(bootClasspath, expectedSuffix) {
		t.Errorf("bootclasspath of %q must start with --system and end with %q, but was %#v.", moduleName, expectedSuffix, bootClasspath)
	}
}

func TestAidlFlagsArePassedToTheAidlCompiler(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["aidl/foo/IFoo.aidl"],
			aidl: { flags: ["-Werror"], },
		}
	`)

	aidlCommand := ctx.ModuleForTests("foo", "android_common").Rule("aidl").RuleParams.Command
	expectedAidlFlag := "-Werror"
	if !strings.Contains(aidlCommand, expectedAidlFlag) {
		t.Errorf("aidl command %q does not contain %q", aidlCommand, expectedAidlFlag)
	}
}

func TestDataNativeBinaries(t *testing.T) {
	ctx, _ := testJava(t, `
		java_test_host {
			name: "foo",
			srcs: ["a.java"],
			data_native_bins: ["bin"]
		}

		python_binary_host {
			name: "bin",
			srcs: ["bin.py"],
		}
	`)

	buildOS := android.BuildOs.String()

	test := ctx.ModuleForTests("foo", buildOS+"_common").Module().(*TestHost)
	entries := android.AndroidMkEntriesForTest(t, ctx, test)[0]
	expected := []string{buildDir + "/.intermediates/bin/" + buildOS + "_x86_64_PY3/bin:bin"}
	actual := entries.EntryMap["LOCAL_COMPATIBILITY_SUPPORT_FILES"]
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected test data - expected: %q, actual: %q", expected, actual)
	}
}
