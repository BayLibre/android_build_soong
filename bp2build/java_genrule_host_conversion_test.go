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

package bp2build

import (
	"testing"

	"android/soong/android"
	"android/soong/java"
)

var otherJavaGenruleHostBp = map[string]string{
	"other/Android.bp": `java_genrule_host {
    name: "foo.tool",
    out: ["foo_tool.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
}
java_genrule_host {
    name: "other.tool",
    out: ["other_tool.out"],
    srcs: ["other_tool.in"],
    cmd: "cp $(in) $(out)",
}`,
}

func runJavaGenruleHostTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	(&tc).moduleTypeUnderTest = "java_genrule_host"
	(&tc).moduleTypeUnderTestFactory = java.GenRuleFactoryHost
	runBp2BuildTestCase(t, func(ctx android.RegistrationContext) {
		ctx.RegisterModuleType("java_genrule", java.GenRuleFactory)
	}, tc)
}

func TestJavaGenruleHostCliVariableReplacement(t *testing.T) {
	runJavaGenruleHostTestCase(t, bp2buildTestCase{
		description: "java_genrule_host with command line variable replacements",
		blueprint: `java_genrule_host {
    name: "foo.tool",
    out: ["foo_tool.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
}

java_genrule_host {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tool"],
    cmd: "$(location :foo.tool) --genDir=$(genDir) arg $(in) $(out)",
}`,
		expectedBazelTargets: []string{
			makeBazelTarget("genrule", "foo", attrNameToString{
				"cmd":   `"$(location :foo.tool) --genDir=$(RULEDIR) arg $(SRCS) $(OUTS)"`,
				"outs":  `["foo.out"]`,
				"srcs":  `["foo.in"]`,
				"tools": `[":foo.tool"]`,
			}),
			makeBazelTarget("genrule", "foo.tool", attrNameToString{
				"cmd":  `"cp $(SRCS) $(OUTS)"`,
				"outs": `["foo_tool.out"]`,
				"srcs": `["foo_tool.in"]`,
			}),
		},
	})
}

func TestJavaGenruleHostUsingLocationsLabel(t *testing.T) {
	runJavaGenruleHostTestCase(t, bp2buildTestCase{
		description: "java_genrule_host using $(locations :label)",
		blueprint: `java_genrule_host {
    name: "foo.tools",
    out: ["foo_tool.out", "foo_tool2.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
}

java_genrule_host {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tools"],
    cmd: "$(locations :foo.tools) -s $(out) $(in)",
}`,
		expectedBazelTargets: []string{
			makeBazelTarget("genrule", "foo", attrNameToString{
				"cmd":   `"$(locations :foo.tools) -s $(OUTS) $(SRCS)"`,
				"outs":  `["foo.out"]`,
				"srcs":  `["foo.in"]`,
				"tools": `[":foo.tools"]`,
			}),
			makeBazelTarget("genrule", "foo.tools", attrNameToString{
				"cmd": `"cp $(SRCS) $(OUTS)"`,
				"outs": `[
        "foo_tool.out",
        "foo_tool2.out",
    ]`,
				"srcs": `["foo_tool.in"]`,
			}),
		},
	})
}

func TestJavaGenruleHostUsingLocationsAbsoluteLabel(t *testing.T) {
	runJavaGenruleHostTestCase(t, bp2buildTestCase{
		description: "java_genrule_host using $(locations //absolute:label)",
		blueprint: `java_genrule_host {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tool_files: [":foo.tool"],
    cmd: "$(locations :foo.tool) -s $(out) $(in)",
}`,
		filesystem: otherJavaGenruleBp,
		expectedBazelTargets: []string{
			makeBazelTarget("genrule", "foo", attrNameToString{
				"cmd":   `"$(locations //other:foo.tool) -s $(OUTS) $(SRCS)"`,
				"outs":  `["foo.out"]`,
				"srcs":  `["foo.in"]`,
				"tools": `["//other:foo.tool"]`,
			}),
		},
	})
}

func TestJavaGenruleHostSrcsUsingAbsoluteLabel(t *testing.T) {
	runJavaGenruleHostTestCase(t, bp2buildTestCase{
		description: "java_genrule_host srcs using $(locations //absolute:label)",
		blueprint: `java_genrule_host {
    name: "foo",
    out: ["foo.out"],
    srcs: [":other.tool"],
    tool_files: [":foo.tool"],
    cmd: "$(locations :foo.tool) -s $(out) $(location :other.tool)",
}`,
		filesystem: otherJavaGenruleBp,
		expectedBazelTargets: []string{
			makeBazelTarget("genrule", "foo", attrNameToString{
				"cmd":   `"$(locations //other:foo.tool) -s $(OUTS) $(location //other:other.tool)"`,
				"outs":  `["foo.out"]`,
				"srcs":  `["//other:other.tool"]`,
				"tools": `["//other:foo.tool"]`,
			}),
		},
	})
}

func TestJavaGenruleHostLocationsLabelUsesFirstToolFile(t *testing.T) {
	runJavaGenruleHostTestCase(t, bp2buildTestCase{
		description: "java_genrule_host using $(location) label should substitute first tool label automatically",
		blueprint: `java_genrule_host {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tool_files: [":foo.tool", ":other.tool"],
    cmd: "$(location) -s $(out) $(in)",
}`,
		filesystem: otherJavaGenruleBp,
		expectedBazelTargets: []string{
			makeBazelTarget("genrule", "foo", attrNameToString{
				"cmd":  `"$(location //other:foo.tool) -s $(OUTS) $(SRCS)"`,
				"outs": `["foo.out"]`,
				"srcs": `["foo.in"]`,
				"tools": `[
        "//other:foo.tool",
        "//other:other.tool",
    ]`,
			}),
		},
	})
}

func TestJavaGenruleHostLocationsLabelUsesFirstTool(t *testing.T) {
	runJavaGenruleHostTestCase(t, bp2buildTestCase{
		description: "java_genrule_host using $(locations) label should substitute first tool label automatically",
		blueprint: `java_genrule_host {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tool", ":other.tool"],
    cmd: "$(locations) -s $(out) $(in)",
}`,
		filesystem: otherJavaGenruleBp,
		expectedBazelTargets: []string{
			makeBazelTarget("genrule", "foo", attrNameToString{
				"cmd":  `"$(locations //other:foo.tool) -s $(OUTS) $(SRCS)"`,
				"outs": `["foo.out"]`,
				"srcs": `["foo.in"]`,
				"tools": `[
        "//other:foo.tool",
        "//other:other.tool",
    ]`,
			}),
		},
	})
}

func TestJavaGenruleHostWithoutToolsOrToolFiles(t *testing.T) {
	runJavaGenruleHostTestCase(t, bp2buildTestCase{
		description: "java_genrule_host without tools or tool_files can convert successfully",
		blueprint: `java_genrule_host {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    cmd: "cp $(in) $(out)",
}`,
		expectedBazelTargets: []string{
			makeBazelTarget("genrule", "foo", attrNameToString{
				"cmd":  `"cp $(SRCS) $(OUTS)"`,
				"outs": `["foo.out"]`,
				"srcs": `["foo.in"]`,
			}),
		},
	})
}
