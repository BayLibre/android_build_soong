package bp2build

import (
	"testing"

	"android/soong/cc"
)

func TestStaticPrebuiltLibrary(t *testing.T) {
	runBp2BuildTestCaseSimple(t,
		bp2buildTestCase{
			description:                "prebuilt library static simple",
			moduleTypeUnderTest:        "cc_prebuilt_library_static",
			moduleTypeUnderTestFactory: cc.PrebuiltStaticLibraryFactory,
			filesystem: map[string]string{
				"libf.so": "",
			},
			blueprint: `
cc_prebuilt_library_static {
	name: "libtest",
	srcs: ["libf.so"],
	bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{
				makeBazelTarget("prebuilt_library_static", "libtest", attrNameToString{
					"static_library": `"libf.so"`,
				}),
			},
		})
}

func TestStaticPrebuiltLibraryWithArchVariance(t *testing.T) {
	runBp2BuildTestCaseSimple(t,
		bp2buildTestCase{
			description:                "prebuilt library static with arch variance",
			moduleTypeUnderTest:        "cc_prebuilt_library_static",
			moduleTypeUnderTestFactory: cc.PrebuiltStaticLibraryFactory,
			filesystem: map[string]string{
				"libf.so": "",
				"libg.so": "",
			},
			blueprint: `
cc_prebuilt_library_static {
	name: "libtest",
	arch: {
		arm64: { srcs: ["libf.so"], },
		arm: { srcs: ["libg.so"], },
	},
	bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{
				makeBazelTarget("prebuilt_library_static", "libtest", attrNameToString{
					"static_library": `select({
        "//build/bazel/platforms/arch:arm": "libg.so",
        "//build/bazel/platforms/arch:arm64": "libf.so",
        "//conditions:default": None,
    })`,
				}),
			},
		})
}
