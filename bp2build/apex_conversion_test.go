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
	"android/soong/android"
	"android/soong/apex"
	"android/soong/cc"

	"testing"
)

func runApexTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	runBp2BuildTestCase(t, registerApexModuleTypes, tc)
}

func registerApexModuleTypes(ctx android.RegistrationContext) {
	// CC module types needed as they can be APEX dependencies viathe  native_shared_libs property
	cc.RegisterCCBuildComponents(ctx)
	ctx.RegisterModuleType("cc_library", cc.LibraryFactory)
}

func TestApexBundleSimple(t *testing.T) {
	runApexTestCase(t, bp2buildTestCase{
		description:                        "apex - simple example",
		moduleTypeUnderTest:                "apex",
		moduleTypeUnderTestFactory:         apex.BundleFactory,
		moduleTypeUnderTestBp2BuildMutator: apex.ApexBundleBp2Build,
		filesystem:                         map[string]string{},
		blueprint: `
cc_library {
        name: "native_shared_lib_1",
	bazel_module: { bp2build_available: false },
}

cc_library {
        name: "native_shared_lib_2",
	bazel_module: { bp2build_available: false },
}

apex {
	name: "com.android.apogee",
	manifest: "manifest.json",
	key: "com.android.key",
	updatable: false,
	installable: false,
	native_shared_libs: [
	    "native_shared_lib_1",
	    "native_shared_lib_2",
	],
	binaries: [
            "binary_1",
	    "binary_2",
	],
}
`,
		expectedBazelTargets: []string{`apex(
    name = "com.android.apogee",
    binaries = [
        "binary_1",
        "binary_2",
    ],
    installable = False,
    key = "com.android.key",
    manifest = "manifest.json",
    native_shared_libs = [
        ":native_shared_lib_1",
        ":native_shared_lib_2",
    ],
    updatable = False,
)`}})
}

func TestApexBundleHasBazelModuleProps(t *testing.T) {
	runApexTestCase(t, bp2buildTestCase{
		description:                        "apex - has bazel module props",
		moduleTypeUnderTest:                "apex",
		moduleTypeUnderTestFactory:         apex.BundleFactory,
		moduleTypeUnderTestBp2BuildMutator: apex.ApexBundleBp2Build,
		filesystem:                         map[string]string{},
		blueprint: `
apex {
	name: "apogee",
	manifest: "manifest.json",
	bazel_module: { bp2build_available: true },
}
`,
		expectedBazelTargets: []string{`apex(
    name = "apogee",
    manifest = "manifest.json",
)`}})
}
