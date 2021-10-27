package bp2build

import (
	"testing"

	"android/soong/android"
	"android/soong/python"
)

func runBp2BuildTestCaseWithLibs(t *testing.T, tc Bp2BuildTestCase) {
	RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {
		ctx.RegisterModuleType("python_library", python.PythonLibraryFactory)
		ctx.RegisterModuleType("python_library_host", python.PythonLibraryHostFactory)
	}, tc)
}

func TestPythonBinaryHostSimple(t *testing.T) {
	runBp2BuildTestCaseWithLibs(t, Bp2BuildTestCase{
		Description:                        "simple python_binary_host converts to a native py_binary",
		ModuleTypeUnderTest:                "python_binary_host",
		ModuleTypeUnderTestFactory:         python.PythonBinaryHostFactory,
		ModuleTypeUnderTestBp2BuildMutator: python.PythonBinaryBp2Build,
		Filesystem: map[string]string{
			"a.py":           "",
			"b/c.py":         "",
			"b/d.py":         "",
			"b/e.py":         "",
			"files/data.txt": "",
		},
		Blueprint: `python_binary_host {
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
      bazel_module: { bp2build_available: true },
    }`,
		ExpectedBazelTargets: []string{`py_binary(
    name = "foo",
    data = ["files/data.txt"],
    deps = [":bar"],
    main = "a.py",
    srcs = [
        "a.py",
        "b/c.py",
        "b/d.py",
    ],
)`,
		},
	})
}

func TestPythonBinaryHostPy2(t *testing.T) {
	RunBp2BuildTestCaseSimple(t, Bp2BuildTestCase{
		Description:                        "py2 python_binary_host",
		ModuleTypeUnderTest:                "python_binary_host",
		ModuleTypeUnderTestFactory:         python.PythonBinaryHostFactory,
		ModuleTypeUnderTestBp2BuildMutator: python.PythonBinaryBp2Build,
		Blueprint: `python_binary_host {
    name: "foo",
    srcs: ["a.py"],
    version: {
        py2: {
            enabled: true,
        },
        py3: {
            enabled: false,
        },
    },

    bazel_module: { bp2build_available: true },
}
`,
		ExpectedBazelTargets: []string{`py_binary(
    name = "foo",
    python_version = "PY2",
    srcs = ["a.py"],
)`,
		},
	})
}

func TestPythonBinaryHostPy3(t *testing.T) {
	RunBp2BuildTestCaseSimple(t, Bp2BuildTestCase{
		Description:                        "py3 python_binary_host",
		ModuleTypeUnderTest:                "python_binary_host",
		ModuleTypeUnderTestFactory:         python.PythonBinaryHostFactory,
		ModuleTypeUnderTestBp2BuildMutator: python.PythonBinaryBp2Build,
		Blueprint: `python_binary_host {
    name: "foo",
    srcs: ["a.py"],
    version: {
        py2: {
            enabled: false,
        },
        py3: {
            enabled: true,
        },
    },

    bazel_module: { bp2build_available: true },
}
`,
		ExpectedBazelTargets: []string{
			// python_version is PY3 by default.
			`py_binary(
    name = "foo",
    srcs = ["a.py"],
)`,
		},
	})
}

func TestPythonBinaryHostArchVariance(t *testing.T) {
	RunBp2BuildTestCaseSimple(t, Bp2BuildTestCase{
		Description:                        "test arch variants",
		ModuleTypeUnderTest:                "python_binary_host",
		ModuleTypeUnderTestFactory:         python.PythonBinaryHostFactory,
		ModuleTypeUnderTestBp2BuildMutator: python.PythonBinaryBp2Build,
		Filesystem: map[string]string{
			"dir/arm.py": "",
			"dir/x86.py": "",
		},
		Blueprint: `python_binary_host {
					 name: "foo-arm",
					 arch: {
						 arm: {
							 srcs: ["arm.py"],
						 },
						 x86: {
							 srcs: ["x86.py"],
						 },
					},
				 }`,
		ExpectedBazelTargets: []string{
			`py_binary(
    name = "foo-arm",
    srcs = select({
        "//build/bazel/platforms/arch:arm": ["arm.py"],
        "//build/bazel/platforms/arch:x86": ["x86.py"],
        "//conditions:default": [],
    }),
)`,
		},
	})
}
