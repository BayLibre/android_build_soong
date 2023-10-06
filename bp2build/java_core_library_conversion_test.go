// Copyright 2023 Google Inc. All rights reserved.
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
)

func TestJavaCoreLibrary(t *testing.T) {
	runJavaLibraryTestCase(t, Bp2buildTestCase{
		Description:             "java core library",
		StubbedBuildDefinitions: []string{"bar"},
		Blueprint: `
java_library {
	name: "bar",
	sdk_version: "none",
}
java_library {
	name: "foo",
	srcs: ["foo.java"],
	patch_module: "java.base",
	system_modules: "bar",
	sdk_version: "none",
	java_version: "1.8",
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("java_core_library", "foo", AttrNameToString{
				"srcs":           `["foo.java"]`,
				"patch_module":   `"java.base"`,
				"system_modules": `":bar"`,
				"java_version":   `"1.8"`,
			}),
			MakeNeverlinkDuplicateTargetWithAttrs("java_core_library",
				"foo",
				AttrNameToString{
					"java_version": `"1.8"`,
				}),
		},
	})
}

func TestMinimalJavaCoreLibrary(t *testing.T) {
	runJavaLibraryTestCase(t, Bp2buildTestCase{
		Description: "java minimal core library",
		Blueprint: `
java_library {
	name: "foo",
	srcs: ["foo.java"],
	system_modules: "none",
	sdk_version: "none",
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("java_core_library", "foo", AttrNameToString{
				"srcs": `["foo.java"]`,
			}),
			MakeNeverlinkDuplicateTargetWithAttrs("java_core_library", "foo", AttrNameToString{}),
		},
	})
}
