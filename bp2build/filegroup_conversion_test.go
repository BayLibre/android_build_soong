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
	"fmt"
	"testing"

	"android/soong/android"
)

func runFilegroupTestCase(t *testing.T, tc Bp2buildTestCase) {
	t.Helper()
	(&tc).ModuleTypeUnderTest = "filegroup"
	(&tc).ModuleTypeUnderTestFactory = android.FileGroupFactory
	RunBp2BuildTestCase(t, registerFilegroupModuleTypes, tc)
}

func registerFilegroupModuleTypes(ctx android.RegistrationContext) {}

func TestFilegroupSameNameAsFile_OneFile(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description: "filegroup - same name as file, with one file",
		Filesystem:  map[string]string{},
		Blueprint: `
filegroup {
    name: "foo",
    srcs: ["foo"],
}
`,
		ExpectedBazelTargets:       []string{},
		ExpectedHandcraftedModules: []string{"foo"}},
	)
}

func TestFilegroupSameNameAsFile_MultipleFiles(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description: "filegroup - same name as file, with multiple files",
		Filesystem:  map[string]string{},
		Blueprint: `
filegroup {
	name: "foo",
	srcs: ["foo", "bar"],
}
`,
		ExpectedErr: fmt.Errorf("filegroup 'foo' cannot contain a file with the same name"),
	})
}

func TestFilegroupWithAidlSrcs(t *testing.T) {
	testcases := []struct {
		name               string
		bp                 string
		expectedBazelAttrs AttrNameToString
	}{
		{
			name: "filegroup with only aidl srcs",
			bp: `
	filegroup {
		name: "foo",
		srcs: ["aidl/foo.aidl"],
		path: "aidl",
	}`,
			expectedBazelAttrs: AttrNameToString{
				"srcs":                `["aidl/foo.aidl"]`,
				"strip_import_prefix": `"aidl"`,
				"tags":                `["apex_available=//apex_available:anyapex"]`,
			},
		},
		{
			name: "filegroup without path",
			bp: `
	filegroup {
		name: "foo",
		srcs: ["aidl/foo.aidl"],
	}`,
			expectedBazelAttrs: AttrNameToString{
				"srcs": `["aidl/foo.aidl"]`,
				"tags": `["apex_available=//apex_available:anyapex"]`,
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			expectedBazelTargets := []string{
				MakeBazelTargetNoRestrictions("aidl_library", "foo", test.expectedBazelAttrs),
			}
			runFilegroupTestCase(t, Bp2buildTestCase{
				Description:          test.name,
				Blueprint:            test.bp,
				ExpectedBazelTargets: expectedBazelTargets,
			})
		})
	}
}

func TestFilegroupWithAidlAndNonAidlSrcs(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description: "filegroup with aidl and non-aidl srcs",
		Filesystem:  map[string]string{},
		Blueprint: `
filegroup {
    name: "foo",
    srcs: [
		"aidl/foo.aidl",
		"buf.proto",
	],
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("filegroup", "foo", AttrNameToString{
				"srcs": `[
        "aidl/foo.aidl",
        "buf.proto",
    ]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_aidl_filegroup", AttrNameToString{
				"srcs": `["aidl/foo.aidl"]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_proto_filegroup", AttrNameToString{
				"srcs": `["buf.proto"]`}),
		}})
}

func TestFilegroupWithProtoSrcs(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description: "filegroup with proto and non-proto srcs",
		Filesystem:  map[string]string{},
		Blueprint: `
filegroup {
		name: "foo",
		srcs: ["proto/foo.proto"],
		path: "proto",
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("proto_library", "foo_proto", AttrNameToString{
				"srcs":                `["proto/foo.proto"]`,
				"strip_import_prefix": `"proto"`,
				"tags": `[
        "apex_available=//apex_available:anyapex",
        "manual",
    ]`,
			}),
			MakeBazelTargetNoRestrictions("alias", "foo_bp2build_converted", AttrNameToString{
				"actual": `"//.:foo_proto"`,
				"tags": `[
        "apex_available=//apex_available:anyapex",
        "manual",
    ]`,
			}),
			MakeBazelTargetNoRestrictions("filegroup", "foo", AttrNameToString{
				"srcs": `["proto/foo.proto"]`}),
		}})
}

func TestFilegroupWithProtoAndNonProtoSrcs(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description: "filegroup with proto and non-proto srcs",
		Filesystem:  map[string]string{},
		Blueprint: `
filegroup {
    name: "foo",
    srcs: [
		"foo.proto",
		"buf.cpp",
	],
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("filegroup", "foo", AttrNameToString{
				"srcs": `[
        "foo.proto",
        "buf.cpp",
    ]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_proto_filegroup", AttrNameToString{
				"srcs": `["foo.proto"]`}),
		}})
}

func TestFilegroupWithProtoInDifferentPackage(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description: "filegroup with .proto in different package",
		Filesystem: map[string]string{
			"subdir/Android.bp": "",
		},
		Blueprint: `
filegroup {
    name: "foo",
    srcs: ["subdir/foo.proto"],
}`,
		Dir: "subdir", // check in subdir
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("proto_library", "foo_proto", AttrNameToString{
				"srcs":                `["//subdir:foo.proto"]`,
				"import_prefix":       `"subdir"`,
				"strip_import_prefix": `""`,
				"tags": `[
        "apex_available=//apex_available:anyapex",
        "manual",
    ]`}),
		}})
}

func TestFilegroupWithVariousSrcs(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description:             "filegroup that has modules and files with various extensions(including .kt) as srcs",
		StubbedBuildDefinitions: []string{"a1_java", "b1_kt", "c1_srcjar", "d1_logtags", "e1_aidl", "f1_proto", "g1_txt"},
		Filesystem:              map[string]string{},
		Blueprint: `
filegroup {
    name: "foo",
    srcs: [
        ":a1_java",
        ":b1_kt",
        ":c1_srcjar",
        ":d1_logtags",
        ":e1_aidl",
        ":f1_proto",
        ":g1_txt",
        "a2.java",
        "b2.kt",
        "c2.srcjar",
        "d2.logtags",
        "e2.aidl",
        "f2.proto",
        "g2.txt",
    ],
}

filegroup {
    name: "a1_java",
    srcs: ["a1.java"],
}

filegroup {
    name: "b1_kt",
    srcs: ["b1.kt"],
}

filegroup {
    name: "c1_srcjar",
    srcs: ["c1.srcjar"],
}

filegroup {
    name: "d1_logtags",
    srcs: ["d1.logtags"],
}

filegroup {
    name: "e1_aidl",
    srcs: ["e1.aidl"],
}

filegroup {
    name: "f1_proto",
    srcs: ["f1.proto"],
}

filegroup {
    name: "g1_txt",
    srcs: ["g1.txt"],
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("filegroup", "foo", AttrNameToString{
				"srcs": `[
        ":a1_java",
        ":b1_kt",
        ":c1_srcjar",
        ":d1_logtags",
        ":e1_aidl",
        ":f1_proto",
        ":g1_txt",
        "a2.java",
        "b2.kt",
        "c2.srcjar",
        "d2.logtags",
        "e2.aidl",
        "f2.proto",
        "g2.txt",
    ]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_kt_jvm_library_filegroup", AttrNameToString{
				"srcs": `[
        "a1.java",
        "a2.java",
        "b1.kt",
        "b2.kt",
        "c1.srcjar",
        "c2.srcjar",
    ]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_logtags_filegroup", AttrNameToString{
				"srcs": `[
        "d1.logtags",
        "d2.logtags",
    ]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_aidl_filegroup", AttrNameToString{
				"srcs": `[
        "e1.aidl",
        "e2.aidl",
    ]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_proto_filegroup", AttrNameToString{
				"srcs": `[
        "f1.proto",
        "f2.proto",
    ]`}),
		}})
}

func TestFilegroupWithoutKtSrcs(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description:             "filegroup that has no .kt in srcs(module or file)",
		StubbedBuildDefinitions: []string{"a1_java"},
		Filesystem:              map[string]string{},
		Blueprint: `
filegroup {
    name: "foo",
    srcs: [
        ":a1_java",
        "a2.java",
    ],
}

filegroup {
    name: "a1_java",
    srcs: ["a1.java"],
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("filegroup", "foo", AttrNameToString{
				"srcs": `[
        ":a1_java",
        "a2.java",
    ]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_java_library_filegroup", AttrNameToString{
				"srcs": `[
        "a1.java",
        "a2.java",
    ]`}),
		}})
}

func TestFilegroupWithFileExcludeSrcs(t *testing.T) {
	runFilegroupTestCase(t, Bp2buildTestCase{
		Description: "filegroup that has a file as exclude_srcs",
		Filesystem:  map[string]string{},
		Blueprint: `
filegroup {
    name: "foo",
    srcs: [
        "a1.java",
        "a2.java",
    ],
    exclude_srcs: [
        "a1.java",
    ],
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("filegroup", "foo", AttrNameToString{
				"srcs": `["a2.java"]`}),
			MakeBazelTargetNoRestrictions("filegroup", "foo_java_library_filegroup", AttrNameToString{
				"srcs": `["a2.java"]`}),
		}})
}
