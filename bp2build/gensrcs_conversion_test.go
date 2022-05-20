package bp2build

import (
	"android/soong/android"
	"android/soong/genrule"
	"testing"
)

func TestGensrcs(t *testing.T) {
	bp := `
gensrcs {
    name: "external_files",
    srcs: ["input.txt"],
    cmd: "cat $(in) > $(out)",
    output_extension: "out",
    bazel_module: { bp2build_available: true },
}

gensrcs {
    name: "foo",
    srcs: ["test/input.txt", ":external_files"],
    tool_files: ["program.py"],
    cmd: "$(location program.py) $(in) $(out)",
    output_extension: "out",
    bazel_module: { bp2build_available: true },
}`
	externalFilesAttrs := attrNameToString{
		"srcs":             `["input.txt"]`,
		"output_extension": `"out"`,
		"cmd":              `"cat $(SRC) > $(OUT)"`,
	}
	fooAttrs := attrNameToString{
		"srcs": `[
        "test/input.txt",
        ":external_files",
    ]`,
		"tools":            `["program.py"]`,
		"output_extension": `"out"`,
		"cmd":              `"$(location program.py) $(SRC) $(OUT)"`,
	}
	expectedBazelTargets := []string{
		makeBazelTarget("gensrcs", "external_files", externalFilesAttrs),
		makeBazelTarget("gensrcs", "foo", fooAttrs),
	}
	t.Run("gensrcs", func(t *testing.T) {
		runBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
			bp2buildTestCase{
				moduleTypeUnderTest:        "gensrcs",
				moduleTypeUnderTestFactory: genrule.GenSrcsFactory,
				blueprint:                  bp,
				expectedBazelTargets:       expectedBazelTargets,
			})
	})
}
