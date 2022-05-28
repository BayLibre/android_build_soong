package bp2build

import (
	"android/soong/android"
	"android/soong/genrule"
	"testing"
)

func TestGensrcs(t *testing.T) {
	testcases := []struct {
		name               string
		bp                 string
		expectedBazelAttrs attrNameToString
	}{
		{
			name: "gensrcs with common usage of properties",
			bp: `
			gensrcs {
                name: "foo",
                srcs: ["test/input.txt", ":external_files"],
                tool_files: ["program.py"],
                cmd: "$(location program.py) $(in) $(out)",
                output_extension: "out",
                bazel_module: { bp2build_available: true },
			}`,
			expectedBazelAttrs: attrNameToString{
				"srcs": `[
        "test/input.txt",
        ":external_files__BP2BUILD__MISSING__DEP",
    ]`,
				"tools":            `["program.py"]`,
				"output_extension": `"out"`,
				"cmd":              `"$(location program.py) $(SRC) $(OUT)"`,
			},
		},
		{
			name: "gensrcs with out_extension unset",
			bp: `
			gensrcs {
                name: "foo",
                srcs: ["input.txt"],
                cmd: "cat $(in) > $(out)",
                bazel_module: { bp2build_available: true },
			}`,
			expectedBazelAttrs: attrNameToString{
				"srcs": `["input.txt"]`,
				"cmd":  `"cat $(SRC) > $(OUT)"`,
			},
		},
	}

	for _, test := range testcases {
		expectedBazelTargets := []string{
			makeBazelTarget("gensrcs", "foo", test.expectedBazelAttrs),
		}
		t.Run(test.name, func(t *testing.T) {
			runBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				bp2buildTestCase{
					moduleTypeUnderTest:        "gensrcs",
					moduleTypeUnderTestFactory: genrule.GenSrcsFactory,
					blueprint:                  test.bp,
					expectedBazelTargets:       expectedBazelTargets,
				})
		})
	}
}
