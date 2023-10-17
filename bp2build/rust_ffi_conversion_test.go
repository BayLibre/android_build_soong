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
	"android/soong/android"
	"android/soong/rust"
	"testing"
)

func runRustFfiTestCase(t *testing.T, tc Bp2buildTestCase) {
	t.Helper()
	RunBp2BuildTestCase(t, registerRustFfiModuleTypes, tc)
}

func registerRustFfiModuleTypes(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("rust_ffi_static", rust.RustFFIStaticFactory)
}

func TestRustFfiStatic(t *testing.T) {
	expectedAttrs := AttrNameToString{
		"crate_name": `"foo"`,
		"srcs": `[
        "src/helper.rs",
        "src/lib.rs",
    ]`,
		"crate_features":  `["bah-enabled"]`,
		"edition":         `"2021"`,
		"export_includes": `["include"]`,
	}

	runRustFfiTestCase(t, Bp2buildTestCase{
		Dir:       "external/rust/crates/foo",
		Blueprint: "",
		Filesystem: map[string]string{
			"external/rust/crates/foo/src/lib.rs":    "",
			"external/rust/crates/foo/src/helper.rs": "",
			"external/rust/crates/foo/Android.bp": `
rust_ffi_static {
    name: "libfoo",
    host_supported: true,
    crate_name: "foo",
    srcs: ["src/lib.rs"],
    edition: "2015",
    rustlibs: [
        "bar",
    ],
    include_dirs: [
        "include",
    ],
    bazel_module: { bp2build_available: true },
}
`,
		},
		ExpectedBazelTargets: []string{
			MakeBazelTargetNoRestrictions("rust_library", "libfoo", expectedAttrs),
		},
	},
	)
}
