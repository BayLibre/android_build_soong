package bp2build

import (
	"android/soong/android"
	"android/soong/rust"
	"testing"
)

func runRustLibraryTestCase(t *testing.T, tc Bp2buildTestCase) {
	t.Helper()
	RunBp2BuildTestCase(t, registerRustLibraryModuleTypes, tc)
}

func registerRustLibraryModuleTypes(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("rust_library", rust.RustLibraryFactory)
	ctx.RegisterModuleType("rust_library_host", rust.RustLibraryHostFactory)
}

func TestRustLibrary(t *testing.T) {
	runRustLibraryTestCase(t, Bp2buildTestCase{
		Dir: "external/rust/crates/foo",
		Blueprint: `
rust_library {
	name: "libfoo",
	crate_name: "foo",
	srcs: ["src/lib.rs"],
	edition: "2021",
	features: ["bah-enabled"],
}
rust_library_host {
	name: "libfoo_host",
	crate_name: "foo",
	srcs: ["src/lib.rs"],
	edition: "2021",
	features: ["bah-enabled"],
}
`,
		Filesystem: map[string]string{
			"external/rust/crates/foo/src/lib.rs":    "",
			"external/rust/crates/foo/src/helper.rs": "",
		},
		ExpectedBazelTargets: []string{
			MakeBazelTarget("rust_library", "libfoo", AttrNameToString{
				"crate_name": "foo",
				"srcs": `[
        "src/lib.rs",
        "src/helper.rs",
    ]`,
				"create_features": `["bah-enabled]`,
				"edition":         "2021",
			}),
			MakeBazelTarget("rust_library", "libfoo_host", AttrNameToString{
				"crate_name": "foo",
				"srcs": `[
        "src/lib.rs",
        "src/helper.rs",
    ]`,
				"create_features": `["bah-enabled]`,
				"edition":         "2021",
			}),
		},
	},
	)
}
