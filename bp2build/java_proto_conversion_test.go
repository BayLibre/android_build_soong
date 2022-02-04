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
	"android/soong/java"
)

func runJavaProtoTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	(&tc).moduleTypeUnderTest = "java_library_static"
	(&tc).moduleTypeUnderTestFactory = java.LibraryFactory
	runBp2BuildTestCase(t, func(ctx android.RegistrationContext) {}, tc)
}

func TestJavaProto(t *testing.T) {
	testJavaProtoSpecializedType(t, "nano")
	testJavaProtoSpecializedType(t, "micro")
	testJavaProtoSpecializedType(t, "lite")
}

func testJavaProtoSpecializedType(t *testing.T, typ string) {
	runJavaProtoTestCase(t, bp2buildTestCase{
		description: fmt.Sprintf("java_proto %s", typ),
		blueprint: fmt.Sprintf(`java_library_static {
    name: "java-protos",
    proto: {
        type: "%s",
    },
    srcs: ["a.proto"],
}
`, typ),
		expectedBazelTargets: []string{
			makeBazelTarget("proto_library", "java-protos_proto", attrNameToString{
				"srcs":                `["a.proto"]`,
				"strip_import_prefix": `""`,
			}),
			makeBazelTarget(
				fmt.Sprintf("java_%s_proto_library", typ),
				fmt.Sprintf("java-protos_java_proto_%s", typ),
				attrNameToString{
					"deps": `[":java-protos_proto"]`,
				}),
			makeBazelTarget("java_library", "java-protos", attrNameToString{
				"deps": fmt.Sprintf(`[":java-protos_java_proto_%s"]`, typ),
			}),
		},
	})
}

func TestJavaProtoFull(t *testing.T) {
	runJavaProtoTestCase(t, bp2buildTestCase{
		description: "java_proto",
		blueprint: `java_library_static {
    name: "java-protos",
    proto: {
        type: "full",
    },
    srcs: ["a.proto"],
}
`,
		expectedBazelTargets: []string{
			makeBazelTarget("proto_library", "java-protos_proto", attrNameToString{
				"srcs":                `["a.proto"]`,
				"strip_import_prefix": `""`,
			}),
			makeBazelTarget(
				"java_proto_library",
				"java-protos_java_proto",
				attrNameToString{
					"deps": `[":java-protos_proto"]`,
				}),
			makeBazelTarget("java_library", "java-protos", attrNameToString{
				"deps": `[":java-protos_java_proto"]`,
			}),
		},
	})
}
