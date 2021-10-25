package bp2build

import (
	"testing"

	"android/soong/cc"
)

func TestSharedPrebuiltLibrary(t *testing.T) {
	runBp2BuildTestCaseSimple(t,
		bp2buildTestCase{
			description:                        "prebuilt library shared simple",
			moduleTypeUnderTest:                "cc_prebuilt_library_shared",
			moduleTypeUnderTestFactory:         cc.PrebuiltSharedLibraryFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.PrebuiltLibrarySharedBp2Build,
			filesystem: map[string]string{
				"libf.so": "",
			},
			blueprint: `
cc_prebuilt_library_shared {
	name: "libtest",
	srcs: ["libf.so"],
	strip: {
			none: true,
	},
	bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{
				`prebuilt_library_shared(
    name = "libtest",
    shared_library = "libf.so",
)`,
			},
		})
}
