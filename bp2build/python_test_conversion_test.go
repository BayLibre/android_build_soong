package bp2build

import (
	"android/soong/python"
	"testing"
)

func TestPythonTestHostSimple(t *testing.T) {
	runBp2BuildTestCaseWithPythonLibraries(t, Bp2buildTestCase{
		Description:                "simple python_test_host converts to a native py_test",
		ModuleTypeUnderTest:        "python_test_host",
		ModuleTypeUnderTestFactory: python.PythonTestHostFactory,
		Filesystem: map[string]string{
			"a.py":           "",
			"b/c.py":         "",
			"b/d.py":         "",
			"b/e.py":         "",
			"files/data.txt": "",
		},
		Blueprint: `python_test_host {
    name: "foo",
    main: "a.py",
    srcs: ["**/*.py"],
    exclude_srcs: ["b/e.py"],
    data: ["files/data.txt",],
    libs: ["bar"],
    bazel_module: { bp2build_available: true },
}
    python_library_host {
      name: "bar",
      srcs: ["b/e.py"],
      bazel_module: { bp2build_available: false },
    }`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("py_test", "foo", AttrNameToString{
				"data":    `["files/data.txt"]`,
				"deps":    `[":bar"]`,
				"main":    `"a.py"`,
				"imports": `["."]`,
				"srcs": `[
        "a.py",
        "b/c.py",
        "b/d.py",
    ]`,
				"target_compatible_with": `select({
        "//build/bazel/platforms/os:android": ["@platforms//:incompatible"],
        "//conditions:default": [],
    })`,
			}),
		},
	})
}
