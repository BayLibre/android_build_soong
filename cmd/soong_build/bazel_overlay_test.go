package main

import (
	"android/soong/android"
	"io/ioutil"
	"os"
	"testing"
)

var buildDir string

func setUp() {
	var err error
	buildDir, err = ioutil.TempDir("", "bazel_overlay_test")
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

type customModule struct {
	android.ModuleBase
}

func (m *customModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// nothing for now.
}

func customModuleFactory() android.Module {
	module := &customModule{}
	android.InitAndroidModule(module)
	return module
}

func TestGenerateBazelOverlayFromBlueprint(t *testing.T) {
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
    deps = [
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
    deps = [
    ],
    ramdisk = True,
)`,
		},
		{
			bp: `custom {
	name: "foo",
	owner: "custom_owner",
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    deps = [
    ],
    owner = "custom_owner",
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
    deps = [
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
    deps = [
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
	dists: [
		{
			targets: ["my_goal"],
			tag: ".tag",
		},
	],
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    deps = [
    ],
    dists = [
        {
            "targets": [
                "my_goal",
            ],
            "tag": ".tag",
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
			targets: ["my_goal"],
			tag: ".tag",
		},
	],
}
		`,
			expectedBazelTarget: `soong_module(
    name = "foo",
    module_name = "foo",
    module_type = "custom",
    module_variant = "",
    deps = [
    ],
    dists = [
        {
            "targets": [
                "my_goal",
            ],
            "tag": ".tag",
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
		ctx := android.NewTestContext()
		ctx.RegisterModuleType("custom", customModuleFactory)
		ctx.Register(config)

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
