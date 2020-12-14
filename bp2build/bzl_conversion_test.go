// Copyright 2020 Google Inc. All rights reserved.
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

package bp2build

import (
	"android/soong/android"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var buildDir string

func setUp() {
	var err error
	buildDir, err = ioutil.TempDir("", "bazel_queryview_test")
	if err != nil {
		panic(err)
	}
}

func tearDown() {
	os.RemoveAll(buildDir)
}

func TestMain(m *testing.M) {
	run := func() int {
		setUp()
		defer tearDown()

		return m.Run()
	}

	os.Exit(run())
}

/*
func TestGenerateBazelQueryViewFromBlueprint(t *testing.T) {
	testCases := []struct {
		bp                  string
		expectedBazelTarget string
	}{
		{
			bp: `custom {
	name: "foo",
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    module_deps = [
    ],
)`,
		},
		{
			bp: `custom {
	name: "foo",
	ramdisk: true,
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    module_deps = [
    ],
    ramdisk = True,
)`,
		},
		{
			bp: `custom {
	name: "foo",
	owner: "a_string_with\"quotes\"_and_\\backslashes\\\\",
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    module_deps = [
    ],
    owner = "a_string_with\"quotes\"_and_\\backslashes\\\\",
)`,
		},
		{
			bp: `custom {
	name: "foo",
	required: ["bar"],
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    module_deps = [
    ],
    required = [
        "bar",
    ],
)`,
		},
		{
			bp: `custom {
	name: "foo",
	target_required: ["qux", "bazqux"],
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    module_deps = [
    ],
    target_required = [
        "qux",
        "bazqux",
    ],
)`,
		},
		{
			bp: `custom {
	name: "foo",
	dist: {
		targets: ["goal_foo"],
		tag: ".foo",
	},
	dists: [
		{
			targets: ["goal_bar"],
			tag: ".bar",
		},
	],
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    module_deps = [
    ],
    dist = {
        "tag": ".foo",
        "targets": [
            "goal_foo",
        ],
    },
    dists = [
        {
            "tag": ".bar",
            "targets": [
                "goal_bar",
            ],
        },
    ],
)`,
		},
		{
			bp: `custom {
	name: "foo",
	required: ["bar"],
	target_required: ["qux", "bazqux"],
	ramdisk: true,
	owner: "custom_owner",
	dists: [
		{
			tag: ".tag",
			targets: ["my_goal"],
		},
	],
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    module_deps = [
    ],
    dists = [
        {
            "tag": ".tag",
            "targets": [
                "my_goal",
            ],
        },
    ],
    owner = "custom_owner",
    ramdisk = True,
    required = [
        "bar",
    ],
    target_required = [
        "qux",
        "bazqux",
    ],
)`,
		},
	}

	for _, testCase := range testCases {
		config := android.TestConfig(buildDir, nil, testCase.bp, nil)
		ctx := android.NewTestContext(config)
		ctx.RegisterModuleType("custom", customModuleFactory)
		ctx.Register()

		_, errs := ctx.ParseFileList(".", []string{"Android.bp"})
		android.FailIfErrored(t, errs)
		_, errs = ctx.PrepareBuildActions(config)
		android.FailIfErrored(t, errs)

		module := ctx.ModuleForTests("foo", "").Module().(*customModule)
		blueprintCtx := ctx.Context.Context

		actualBazelTarget := generateSoongModuleTarget(blueprintCtx, module)
		if actualBazelTarget != testCase.expectedBazelTarget {
			t.Errorf(
				"Expected generated Bazel target to be '%s', got '%s'",
				testCase.expectedBazelTarget,
				actualBazelTarget,
			)
		}
	}
}
*/

func TestGenerateModuleRuleShims(t *testing.T) {
	moduleTypeFactories := map[string]android.ModuleFactory{
		"custom":          customModuleFactoryBase,
		"custom_test":     customTestModuleFactoryBase,
		"custom_defaults": customDefaultsModuleFactoryBasic,
	}
	ruleShims := CreateRuleShims(moduleTypeFactories)

	if len(ruleShims) != 1 {
		t.Errorf("Expected to generate 1 rule shim, but got %d", len(ruleShims))
	}

	ruleShim := ruleShims["bp2build"]
	expectedRules := []string{
		"custom",
		"custom_defaults",
		"custom_test_",
	}

	if len(ruleShim.rules) != len(expectedRules) {
		t.Errorf("Expected %d rules, but got %d", len(expectedRules), len(ruleShim.rules))
	}

	for i, rule := range ruleShim.rules {
		if rule != expectedRules[i] {
			t.Errorf("Expected rule shim to contain %s, but got %s", expectedRules[i], rule)
		}
	}
	expectedBzl := `load("@bp2build//build/bazel/rules:providers.bzl", "SoongModuleInfo")

def _custom_impl(ctx):
    return [SoongModuleInfo()]

custom = rule(
    implementation = _custom_impl,
    attrs = {
        "module_name": attr.string(mandatory = True),
        "module_variant": attr.string(),
        "module_deps": attr.label_list(providers = [SoongModuleInfo]),
        "bool_prop": attr.bool(),
        "bool_ptr_prop": attr.bool(),
        "int64_ptr_prop": attr.int(),
        # nested_props start
#         "nested_prop": attr.string(),
        # nested_props end
        # nested_props_ptr start
#         "nested_prop": attr.string(),
        # nested_props_ptr end
        "string_list_prop": attr.string_list(),
        "string_prop": attr.string(),
        "string_ptr_prop": attr.string(),
    },
)

def _custom_defaults_impl(ctx):
    return [SoongModuleInfo()]

custom_defaults = rule(
    implementation = _custom_defaults_impl,
    attrs = {
        "module_name": attr.string(mandatory = True),
        "module_variant": attr.string(),
        "module_deps": attr.label_list(providers = [SoongModuleInfo]),
        "bool_prop": attr.bool(),
        "bool_ptr_prop": attr.bool(),
        "int64_ptr_prop": attr.int(),
        # nested_props start
#         "nested_prop": attr.string(),
        # nested_props end
        # nested_props_ptr start
#         "nested_prop": attr.string(),
        # nested_props_ptr end
        "string_list_prop": attr.string_list(),
        "string_prop": attr.string(),
        "string_ptr_prop": attr.string(),
    },
)

def _custom_test__impl(ctx):
    return [SoongModuleInfo()]

custom_test_ = rule(
    implementation = _custom_test__impl,
    attrs = {
        "module_name": attr.string(mandatory = True),
        "module_variant": attr.string(),
        "module_deps": attr.label_list(providers = [SoongModuleInfo]),
        "bool_prop": attr.bool(),
        "bool_ptr_prop": attr.bool(),
        "int64_ptr_prop": attr.int(),
        # nested_props start
#         "nested_prop": attr.string(),
        # nested_props end
        # nested_props_ptr start
#         "nested_prop": attr.string(),
        # nested_props_ptr end
        "string_list_prop": attr.string_list(),
        "string_prop": attr.string(),
        "string_ptr_prop": attr.string(),
        # test_prop start
#         "test_string_prop": attr.string(),
        # test_prop end
    },
)
`

	if ruleShim.content != expectedBzl {
		t.Errorf(
			"Expected the generated rule shim bzl to be:\n%s\nbut got:\n%s",
			expectedBzl,
			ruleShim.content)
	}
}

func TestGenerateSoongModuleBzl(t *testing.T) {
	ruleShims := map[string]RuleShim{
		"file1": RuleShim{
			rules:   []string{"a", "b"},
			content: "irrelevant",
		},
		"file2": RuleShim{
			rules:   []string{"c", "d"},
			content: "irrelevant",
		},
	}
	files := CreateBazelFiles(ruleShims, make(map[string][]BazelTarget))

	var actualSoongModuleBzl BazelFile
	for _, f := range files {
		if f.Basename == "soong_module.bzl" {
			actualSoongModuleBzl = f
		}
	}

	expectedLoad := `load("@bp2build//build/bazel/rules:file1.bzl", "a", "b")
load("@bp2build//build/bazel/rules:file2.bzl", "c", "d")
`
	expectedRuleMap := `soong_module_rule_map = {
    "a": a,
    "b": b,
    "c": c,
    "d": d,
}`
	if !strings.Contains(actualSoongModuleBzl.Contents, expectedLoad) {
		t.Errorf(
			"Generated soong_module.bzl:\n\n%s\n\n"+
				"Could not find the load statement in the generated soong_module.bzl:\n%s",
			actualSoongModuleBzl.Contents,
			expectedLoad)
	}

	if !strings.Contains(actualSoongModuleBzl.Contents, expectedRuleMap) {
		t.Errorf(
			"Generated soong_module.bzl:\n\n%s\n\n"+
				"Could not find the module -> rule map in the generated soong_module.bzl:\n%s",
			actualSoongModuleBzl.Contents,
			expectedRuleMap)
	}
}
