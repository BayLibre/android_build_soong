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
	"android/soong/cc"
	"android/soong/genrule"
)

const (
	// See cc/testing.go for more context
	soongCcLibrarySharedPreamble = soongCcLibraryStaticPreamble
)

func registerCcLibrarySharedModuleTypes(ctx android.RegistrationContext) {
	cc.RegisterCCBuildComponents(ctx)
	ctx.RegisterModuleType("toolchain_library", cc.ToolchainLibraryFactory)
	ctx.RegisterModuleType("cc_library_headers", cc.LibraryHeaderFactory)
	ctx.RegisterModuleType("cc_library_static", cc.LibraryStaticFactory)
	ctx.RegisterModuleType("genrule", genrule.GenRuleFactory)
	// Required for system_shared_libs dependencies.
	ctx.RegisterModuleType("cc_library", cc.LibraryFactory)
}

func runCcLibrarySharedTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	runBp2BuildTestCase(t, registerCcLibrarySharedModuleTypes, tc)
}

func TestCcLibrarySharedSimple(t *testing.T) {
	runCcLibrarySharedTestCase(t, bp2buildTestCase{
		description:                        "cc_library_shared test",
		moduleTypeUnderTest:                "cc_library_shared",
		moduleTypeUnderTestFactory:         cc.LibrarySharedFactory,
		moduleTypeUnderTestBp2BuildMutator: cc.CcLibrarySharedBp2Build,
		filesystem: map[string]string{
			// NOTE: include_dir headers *should not* appear in Bazel hdrs later (?)
			"include_dir_1/include_dir_1_a.h": "",
			"include_dir_1/include_dir_1_b.h": "",
			"include_dir_2/include_dir_2_a.h": "",
			"include_dir_2/include_dir_2_b.h": "",
			// NOTE: local_include_dir headers *should not* appear in Bazel hdrs later (?)
			"local_include_dir_1/local_include_dir_1_a.h": "",
			"local_include_dir_1/local_include_dir_1_b.h": "",
			"local_include_dir_2/local_include_dir_2_a.h": "",
			"local_include_dir_2/local_include_dir_2_b.h": "",
			// NOTE: export_include_dir headers *should* appear in Bazel hdrs later
			"export_include_dir_1/export_include_dir_1_a.h": "",
			"export_include_dir_1/export_include_dir_1_b.h": "",
			"export_include_dir_2/export_include_dir_2_a.h": "",
			"export_include_dir_2/export_include_dir_2_b.h": "",
			// NOTE: Soong implicitly includes headers in the current directory
			"implicit_include_1.h": "",
			"implicit_include_2.h": "",
		},
		blueprint: soongCcLibrarySharedPreamble + `
cc_library_headers {
    name: "header_lib_1",
    export_include_dirs: ["header_lib_1"],
}

cc_library_headers {
    name: "header_lib_2",
    export_include_dirs: ["header_lib_2"],
}

cc_library_shared {
    name: "shared_lib_1",
    srcs: ["shared_lib_1.cc"],
}

cc_library_shared {
    name: "shared_lib_2",
    srcs: ["shared_lib_2.cc"],
}

cc_library_static {
    name: "whole_static_lib_1",
    srcs: ["whole_static_lib_1.cc"],
}

cc_library_static {
    name: "whole_static_lib_2",
    srcs: ["whole_static_lib_2.cc"],
}

cc_library_shared {
    name: "foo_shared",
    srcs: [
        "foo_shared1.cc",
        "foo_shared2.cc",
    ],
    cflags: [
        "-Dflag1",
        "-Dflag2"
    ],
    shared_libs: [
        "shared_lib_1",
        "shared_lib_2"
    ],
    whole_static_libs: [
        "whole_static_lib_1",
        "whole_static_lib_2"
    ],
    include_dirs: [
        "include_dir_1",
        "include_dir_2",
    ],
    local_include_dirs: [
        "local_include_dir_1",
        "local_include_dir_2",
    ],
    export_include_dirs: [
        "export_include_dir_1",
        "export_include_dir_2"
    ],
    header_libs: [
        "header_lib_1",
        "header_lib_2"
    ],

    // TODO: Also support export_header_lib_headers
}`,
		expectedBazelTargets: []string{`cc_library_shared(
    name = "foo_shared",
    copts = [
        "-Dflag1",
        "-Dflag2",
        "-Iinclude_dir_1",
        "-I$(BINDIR)/include_dir_1",
        "-Iinclude_dir_2",
        "-I$(BINDIR)/include_dir_2",
        "-Ilocal_include_dir_1",
        "-I$(BINDIR)/local_include_dir_1",
        "-Ilocal_include_dir_2",
        "-I$(BINDIR)/local_include_dir_2",
        "-I.",
        "-I$(BINDIR)/.",
    ],
    implementation_deps = [
        ":header_lib_1",
        ":header_lib_2",
        ":shared_lib_1",
        ":shared_lib_2",
    ],
    includes = [
        "export_include_dir_1",
        "export_include_dir_2",
    ],
    srcs = [
        "foo_shared1.cc",
        "foo_shared2.cc",
    ],
    whole_archive_deps = [
        ":whole_static_lib_1",
        ":whole_static_lib_2",
    ],
)`, `cc_library_shared(
    name = "shared_lib_1",
    copts = [
        "-I.",
        "-I$(BINDIR)/.",
    ],
    srcs = ["shared_lib_1.cc"],
)`, `cc_library_shared(
    name = "shared_lib_2",
    copts = [
        "-I.",
        "-I$(BINDIR)/.",
    ],
    srcs = ["shared_lib_2.cc"],
)`, `cc_library_static(
    name = "whole_static_lib_1",
    copts = [
        "-I.",
        "-I$(BINDIR)/.",
    ],
    srcs = ["whole_static_lib_1.cc"],
)`, `cc_library_static(
    name = "whole_static_lib_2",
    copts = [
        "-I.",
        "-I$(BINDIR)/.",
    ],
    srcs = ["whole_static_lib_2.cc"],
)`},
	})
}
