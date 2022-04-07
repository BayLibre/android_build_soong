package bp2build

import (
	"testing"

	"android/soong/cc"
)

func TestPrebuiltLibraryStaticAndSharedSimple(t *testing.T) {
	runBp2BuildTestCaseSimple(t,
		bp2buildTestCase{
			description:                "prebuilt library static and shared simple",
			moduleTypeUnderTest:        "cc_prebuilt_library",
			moduleTypeUnderTestFactory: cc.PrebuiltLibraryFactory,
			filesystem: map[string]string{
				"libf.so": "",
			},
			blueprint: `
cc_prebuilt_library {
	name: "libtest",
	srcs: ["libf.so"],
	bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{
				makeBazelTarget("prebuilt_library_static", "libtest_bp2build_cc_library_static", attrNameToString{
					"static_library": `"libf.so"`,
				}),
				makeBazelTarget("prebuilt_library_shared", "libtest", attrNameToString{
					"shared_library": `"libf.so"`,
				}),
			},
		})
}

func TestPrebuiltLibraryWithArchVariance(t *testing.T) {
	runBp2BuildTestCaseSimple(t,
		bp2buildTestCase{
			description:                "prebuilt library with arch variance",
			moduleTypeUnderTest:        "cc_prebuilt_library",
			moduleTypeUnderTestFactory: cc.PrebuiltLibraryFactory,
			filesystem: map[string]string{
				"libf.so": "",
				"libg.so": "",
			},
			blueprint: `
cc_prebuilt_library {
	name: "libtest",
	arch: {
		arm64: { srcs: ["libf.so"], },
		arm: { srcs: ["libg.so"], },
	},
	bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{
				makeBazelTarget("prebuilt_library_static", "libtest_bp2build_cc_library_static", attrNameToString{
					"static_library": `select({
        "//build/bazel/platforms/arch:arm": "libg.so",
        "//build/bazel/platforms/arch:arm64": "libf.so",
        "//conditions:default": None,
    })`,
				}),
				makeBazelTarget("prebuilt_library_shared", "libtest", attrNameToString{
					"shared_library": `select({
        "//build/bazel/platforms/arch:arm": "libg.so",
        "//build/bazel/platforms/arch:arm64": "libf.so",
        "//conditions:default": None,
    })`,
				}),
			},
		})
}

func TestPrebuiltLibraryAdditionalAttrs(t *testing.T) {
	runBp2BuildTestCaseSimple(t,
		bp2buildTestCase{
			description:                "prebuilt library additional attributes",
			moduleTypeUnderTest:        "cc_prebuilt_library",
			moduleTypeUnderTestFactory: cc.PrebuiltLibraryFactory,
			filesystem: map[string]string{
				"libf.so":    "",
				"testdir/1/": "",
				"testdir/2/": "",
			},
			blueprint: `
cc_prebuilt_library {
	name: "libtest",
	srcs: ["libf.so"],
	export_include_dirs: ["testdir/1/"],
	export_system_include_dirs: ["testdir/2/"],
	bazel_module: { bp2build_available: true },
}`,
			expectedBazelTargets: []string{
				makeBazelTarget("prebuilt_library_static", "libtest_bp2build_cc_library_static", attrNameToString{
					"static_library":         `"libf.so"`,
					"export_includes":        `["testdir/1/"]`,
					"export_system_includes": `["testdir/2/"]`,
				}),
				makeBazelTarget("prebuilt_library_shared", "libtest", attrNameToString{
					"shared_library": `"libf.so"`,
				}),
			},
		})
}

// TODO(b/228623543): When this bug is fixed, enable this test
//func TestPrebuiltLibraryOnlyShared(t *testing.T) {
//	runBp2BuildTestCaseSimple(t,
//		bp2buildTestCase{
//			description:                "prebuilt library shared only",
//			moduleTypeUnderTest:        "cc_prebuilt_library",
//			moduleTypeUnderTestFactory: cc.PrebuiltLibraryFactory,
//			filesystem: map[string]string{
//				"libf.so": "",
//			},
//			blueprint: `
//cc_prebuilt_library {
//	name: "libtest",
//	srcs: ["libf.so"],
//	static: {
//		enabled: false,
//	},
//	bazel_module: { bp2build_available: true },
//}`,
//			expectedBazelTargets: []string{
//				makeBazelTarget("prebuilt_library_shared", "libtest", attrNameToString{
//					"shared_library": `"libf.so"`,
//				}),
//			},
//		})
//}

// TODO(b/228623543): When this bug is fixed, enable this test
//func TestPrebuiltLibraryOnlyStatic(t *testing.T) {
//	runBp2BuildTestCaseSimple(t,
//		bp2buildTestCase{
//			description:                "prebuilt library static only",
//			moduleTypeUnderTest:        "cc_prebuilt_library",
//			moduleTypeUnderTestFactory: cc.PrebuiltLibraryFactory,
//			filesystem: map[string]string{
//				"libf.so": "",
//			},
//			blueprint: `
//cc_prebuilt_library {
//	name: "libtest",
//	srcs: ["libf.so"],
//	shared: {
//		enabled: false,
//	},
//	bazel_module: { bp2build_available: true },
//}`,
//			expectedBazelTargets: []string{
//				makeBazelTarget("prebuilt_library_static", "libtest_bp2build_cc_library_static", attrNameToString{
//					"static_library": `"libf.so"`,
//				}),
//			},
//		})
//}
