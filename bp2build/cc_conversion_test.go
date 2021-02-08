package bp2build

import (
	"android/soong/android"
	"android/soong/cc"
	"fmt"
	"strings"
	"testing"
)

// Unit tests for converting cc rules with bp2build.

// cc_object tests.
func TestCcObjectBp2Build(t *testing.T) {
	testCases := []struct {
		description                        string
		moduleTypeUnderTest                string
		moduleTypeUnderTestFactory         android.ModuleFactory
		moduleTypeUnderTestBp2BuildMutator func(android.TopDownMutatorContext)
		bp                                 string
		expectedBazelTargets               []string
		fs                                 map[string]string
	}{
		{
			description:                        "simple cc_object generates cc_object with include header dep",
			moduleTypeUnderTest:                "cc_object",
			moduleTypeUnderTestFactory:         cc.ObjectFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.ObjectBp2Build,
			fs: map[string]string{
				"a/b/foo.h": "",
				"a/b/bar.h": "",
				"a/b/c.c":   "",
			},
			bp: `cc_object {
    name: "foo",
    local_include_dirs: ["include"],
    cflags: [
        "-Wno-gcc-compat",
        "-Wall",
        "-Werror",
    ],
    srcs: [
        "a/b/*.h",
        "a/b/c.c"
    ],

    bazel_module: { bp2build_available: true },
}
`,
			expectedBazelTargets: []string{`cc_object(
    name = "foo",
    copts = [
        "-fno-addrsig",
        "-Wno-gcc-compat",
        "-Wall",
        "-Werror",
    ],
    deps = [
        "foo__include",
    ],
    srcs = [
        "a/b/bar.h",
        "a/b/foo.h",
        "a/b/c.c",
    ],
)`,
				`cc_library(
    name = "foo__include",
    hdrs = [
    ],
    includes = [
        "include",
    ],
    strip_include_prefix = "include",
)`,
			},
		},
		{
			description:                        "simple cc_object with defaults",
			moduleTypeUnderTest:                "cc_object",
			moduleTypeUnderTestFactory:         cc.ObjectFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.ObjectBp2Build,
			bp: `cc_object {
    name: "foo",
    local_include_dirs: ["include"],
    srcs: [
        "a/b/*.h",
        "a/b/c.c"
    ],

    defaults: ["foo_defaults"],
    bazel_module: { bp2build_available: true },
}

cc_defaults {
    name: "foo_defaults",
    defaults: ["foo_bar_defaults"],
	// TODO(b/178130668): handle configurable attributes that depend on the platform
    arch: {
        x86: {
            cflags: ["-fPIC"],
        },
        x86_64: {
            cflags: ["-fPIC"],
        },
    },
}

cc_defaults {
    name: "foo_bar_defaults",
    cflags: [
        "-Wno-gcc-compat",
        "-Wall",
        "-Werror",
    ],
}
`,
			expectedBazelTargets: []string{`cc_object(
    name = "foo",
    copts = [
        "-Wno-gcc-compat",
        "-Wall",
        "-Werror",
        "-fno-addrsig",
    ],
    deps = [
        "foo__include",
    ],
    srcs = [
        "a/b/c.c",
    ],
)`,
				`cc_library(
    name = "foo__include",
    hdrs = [
    ],
    includes = [
        "include",
    ],
    strip_include_prefix = "include",
)`,
			},
		},
	}

	dir := "."
	for _, testCase := range testCases {
		fs := make(map[string][]byte)
		toParse := []string{
			"Android.bp",
		}
		for f, content := range testCase.fs {
			if strings.HasSuffix(f, "Android.bp") {
				toParse = append(toParse, f)
			}
			fs[f] = []byte(content)
		}
		config := android.TestConfig(buildDir, nil, testCase.bp, fs)
		ctx := android.NewTestContext(config)
		// Always register cc_defaults module factory
		ctx.RegisterModuleType("cc_defaults", func() android.Module { return cc.DefaultsFactory() })

		ctx.RegisterModuleType(testCase.moduleTypeUnderTest, testCase.moduleTypeUnderTestFactory)
		ctx.RegisterBp2BuildMutator(testCase.moduleTypeUnderTest, testCase.moduleTypeUnderTestBp2BuildMutator)
		ctx.RegisterForBazelConversion()

		_, errs := ctx.ParseFileList(dir, toParse)
		if Errored(t, testCase.description, errs) {
			continue
		}
		_, errs = ctx.ResolveDependencies(config)
		if Errored(t, testCase.description, errs) {
			continue
		}

		bazelTargets := GenerateBazelTargets(ctx.Context.Context, Bp2Build)[dir]
		if actualCount, expectedCount := len(bazelTargets), len(testCase.expectedBazelTargets); actualCount != expectedCount {
			fmt.Println(bazelTargets)
			t.Errorf("%s: Expected %d bazel target, got %d", testCase.description, expectedCount, actualCount)
		} else {
			for i, target := range bazelTargets {
				if w, g := testCase.expectedBazelTargets[i], target.content; w != g {
					t.Errorf(
						"%s: Expected generated Bazel target to be '%s', got '%s'",
						testCase.description,
						w,
						g,
					)
				}
			}
		}
	}
}
