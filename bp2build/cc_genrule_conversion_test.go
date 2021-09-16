package bp2build

import (
	"strings"
	"testing"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/genrule"
)

func TestCcGenruleBp2Build(t *testing.T) {
	otherCcGenruleBp := map[string]string{
		"other/Android.bp": `cc_genrule {
    name: "foo.tool",
    out: ["foo_tool.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
}
cc_genrule {
    name: "other.tool",
    out: ["other_tool.out"],
    srcs: ["other_tool.in"],
    cmd: "cp $(in) $(out)",
}`,
	}

	testCases := []bp2buildTestCase{
		{
			description:                        "cc_genrule with command line variable replacements",
			moduleTypeUnderTest:                "cc_genrule",
			moduleTypeUnderTestFactory:         cc.GenRuleFactory,
			moduleTypeUnderTestBp2BuildMutator: genrule.GenruleBp2Build,
			blueprint: `cc_genrule {
    name: "foo.tool",
    out: ["foo_tool.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
    bazel_module: { bp2build_available: true },
}

cc_genrule {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tool"],
    cmd: "$(location :foo.tool) --genDir=$(genDir) arg $(in) $(out)",
    bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{
				`genrule(
    name = "foo",
    cmd = "$(location :foo.tool) --genDir=$(GENDIR) arg $(SRCS) $(OUTS)",
    outs = ["foo.out"],
    srcs = ["foo.in"],
    tools = [":foo.tool"],
)`,
				`genrule(
    name = "foo.tool",
    cmd = "cp $(SRCS) $(OUTS)",
    outs = ["foo_tool.out"],
    srcs = ["foo_tool.in"],
)`,
			},
		},
		{
			description:                        "cc_genrule using $(locations :label)",
			moduleTypeUnderTest:                "cc_genrule",
			moduleTypeUnderTestFactory:         cc.GenRuleFactory,
			moduleTypeUnderTestBp2BuildMutator: genrule.GenruleBp2Build,
			blueprint: `cc_genrule {
    name: "foo.tools",
    out: ["foo_tool.out", "foo_tool2.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
    bazel_module: { bp2build_available: true },
}

cc_genrule {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tools"],
    cmd: "$(locations :foo.tools) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{`genrule(
    name = "foo",
    cmd = "$(locations :foo.tools) -s $(OUTS) $(SRCS)",
    outs = ["foo.out"],
    srcs = ["foo.in"],
    tools = [":foo.tools"],
)`,
				`genrule(
    name = "foo.tools",
    cmd = "cp $(SRCS) $(OUTS)",
    outs = [
        "foo_tool.out",
        "foo_tool2.out",
    ],
    srcs = ["foo_tool.in"],
)`,
			},
		},
		{
			description:                        "cc_genrule using $(locations //absolute:label)",
			moduleTypeUnderTest:                "cc_genrule",
			moduleTypeUnderTestFactory:         cc.GenRuleFactory,
			moduleTypeUnderTestBp2BuildMutator: genrule.GenruleBp2Build,
			blueprint: `cc_genrule {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tool_files: [":foo.tool"],
    cmd: "$(locations :foo.tool) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{`genrule(
    name = "foo",
    cmd = "$(locations //other:foo.tool) -s $(OUTS) $(SRCS)",
    outs = ["foo.out"],
    srcs = ["foo.in"],
    tools = ["//other:foo.tool"],
)`,
			},
			filesystem: otherCcGenruleBp,
		},
		{
			description:                        "cc_genrule srcs using $(locations //absolute:label)",
			moduleTypeUnderTest:                "cc_genrule",
			moduleTypeUnderTestFactory:         cc.GenRuleFactory,
			moduleTypeUnderTestBp2BuildMutator: genrule.GenruleBp2Build,
			blueprint: `cc_genrule {
    name: "foo",
    out: ["foo.out"],
    srcs: [":other.tool"],
    tool_files: [":foo.tool"],
    cmd: "$(locations :foo.tool) -s $(out) $(location :other.tool)",
    bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{`genrule(
    name = "foo",
    cmd = "$(locations //other:foo.tool) -s $(OUTS) $(location //other:other.tool)",
    outs = ["foo.out"],
    srcs = ["//other:other.tool"],
    tools = ["//other:foo.tool"],
)`,
			},
			filesystem: otherCcGenruleBp,
		},
		{
			description:                        "cc_genrule using $(location) label should substitute first tool label automatically",
			moduleTypeUnderTest:                "cc_genrule",
			moduleTypeUnderTestFactory:         cc.GenRuleFactory,
			moduleTypeUnderTestBp2BuildMutator: genrule.GenruleBp2Build,
			blueprint: `cc_genrule {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tool_files: [":foo.tool", ":other.tool"],
    cmd: "$(location) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{`genrule(
    name = "foo",
    cmd = "$(location //other:foo.tool) -s $(OUTS) $(SRCS)",
    outs = ["foo.out"],
    srcs = ["foo.in"],
    tools = [
        "//other:foo.tool",
        "//other:other.tool",
    ],
)`,
			},
			filesystem: otherCcGenruleBp,
		},
		{
			description:                        "cc_genrule using $(locations) label should substitute first tool label automatically",
			moduleTypeUnderTest:                "cc_genrule",
			moduleTypeUnderTestFactory:         cc.GenRuleFactory,
			moduleTypeUnderTestBp2BuildMutator: genrule.GenruleBp2Build,
			blueprint: `cc_genrule {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tool", ":other.tool"],
    cmd: "$(locations) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{`genrule(
    name = "foo",
    cmd = "$(locations //other:foo.tool) -s $(OUTS) $(SRCS)",
    outs = ["foo.out"],
    srcs = ["foo.in"],
    tools = [
        "//other:foo.tool",
        "//other:other.tool",
    ],
)`,
			},
			filesystem: otherCcGenruleBp,
		},
		{
			description:                        "cc_genrule without tools or tool_files can convert successfully",
			moduleTypeUnderTest:                "cc_genrule",
			moduleTypeUnderTestFactory:         cc.GenRuleFactory,
			moduleTypeUnderTestBp2BuildMutator: genrule.GenruleBp2Build,
			blueprint: `cc_genrule {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    cmd: "cp $(in) $(out)",
    bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{`genrule(
    name = "foo",
    cmd = "cp $(SRCS) $(OUTS)",
    outs = ["foo.out"],
    srcs = ["foo.in"],
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
		for f, content := range testCase.filesystem {
			if strings.HasSuffix(f, "Android.bp") {
				toParse = append(toParse, f)
			}
			fs[f] = []byte(content)
		}
		config := android.TestConfig(buildDir, nil, testCase.blueprint, fs)
		ctx := android.NewTestContext(config)
		ctx.RegisterModuleType(testCase.moduleTypeUnderTest, testCase.moduleTypeUnderTestFactory)
		ctx.RegisterBp2BuildMutator(testCase.moduleTypeUnderTest, testCase.moduleTypeUnderTestBp2BuildMutator)
		ctx.RegisterForBazelConversion()

		_, errs := ctx.ParseFileList(dir, toParse)
		if errored(t, testCase, errs) {
			continue
		}
		_, errs = ctx.ResolveDependencies(config)
		if errored(t, testCase, errs) {
			continue
		}

		checkDir := dir
		if testCase.dir != "" {
			checkDir = testCase.dir
		}

		codegenCtx := NewCodegenContext(config, *ctx.Context, Bp2Build)
		bazelTargets := generateBazelTargetsForDir(codegenCtx, checkDir)
		if actualCount, expectedCount := len(bazelTargets), len(testCase.expectedBazelTargets); actualCount != expectedCount {
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
