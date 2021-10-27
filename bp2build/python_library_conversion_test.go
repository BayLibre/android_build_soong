package bp2build

import (
	"fmt"
	"testing"

	"android/soong/android"
	"android/soong/python"
)

// TODO(alexmarquez): Should be lifted into a generic Bp2Build file
type PythonLibBp2Build func(ctx android.TopDownMutatorContext)

func TestPythonLibrary(t *testing.T) {
	testPythonLib(t, "python_library",
		python.PythonLibraryFactory, python.PythonLibraryBp2Build,
		func(ctx android.RegistrationContext) {})
}

func TestPythonLibraryHost(t *testing.T) {
	testPythonLib(t, "python_library_host",
		python.PythonLibraryHostFactory, python.PythonLibraryHostBp2Build,
		func(ctx android.RegistrationContext) {
			ctx.RegisterModuleType("python_library", python.PythonLibraryFactory)
		})
}

func testPythonLib(t *testing.T, modType string,
	factory android.ModuleFactory, mutator PythonLibBp2Build,
	registration func(ctx android.RegistrationContext)) {
	t.Helper()
	// Simple
	RunBp2BuildTestCase(t, registration, Bp2BuildTestCase{
		Description:                        fmt.Sprintf("simple %s converts to a native py_library", modType),
		ModuleTypeUnderTest:                modType,
		ModuleTypeUnderTestFactory:         factory,
		ModuleTypeUnderTestBp2BuildMutator: mutator,
		Filesystem: map[string]string{
			"a.py":           "",
			"b/c.py":         "",
			"b/d.py":         "",
			"b/e.py":         "",
			"files/data.txt": "",
		},
		Blueprint: fmt.Sprintf(`%s {
    name: "foo",
    srcs: ["**/*.py"],
    exclude_srcs: ["b/e.py"],
    data: ["files/data.txt",],
    libs: ["bar"],
    bazel_module: { bp2build_available: true },
}
    python_library {
      name: "bar",
      srcs: ["b/e.py"],
      bazel_module: { bp2build_available: false },
    }`, modType),
		ExpectedBazelTargets: []string{`py_library(
    name = "foo",
    data = ["files/data.txt"],
    deps = [":bar"],
    srcs = [
        "a.py",
        "b/c.py",
        "b/d.py",
    ],
    srcs_version = "PY3",
)`,
		},
	})

	// PY2
	RunBp2BuildTestCaseSimple(t, Bp2BuildTestCase{
		Description:                        fmt.Sprintf("py2 %s converts to a native py_library", modType),
		ModuleTypeUnderTest:                modType,
		ModuleTypeUnderTestFactory:         factory,
		ModuleTypeUnderTestBp2BuildMutator: mutator,
		Blueprint: fmt.Sprintf(`%s {
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
}`, modType),
		ExpectedBazelTargets: []string{`py_library(
    name = "foo",
    srcs = ["a.py"],
    srcs_version = "PY2",
)`,
		},
	})

	// PY3
	RunBp2BuildTestCaseSimple(t, Bp2BuildTestCase{
		Description:                        fmt.Sprintf("py3 %s converts to a native py_library", modType),
		ModuleTypeUnderTest:                modType,
		ModuleTypeUnderTestFactory:         factory,
		ModuleTypeUnderTestBp2BuildMutator: mutator,
		Blueprint: fmt.Sprintf(`%s {
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
}`, modType),
		ExpectedBazelTargets: []string{`py_library(
    name = "foo",
    srcs = ["a.py"],
    srcs_version = "PY3",
)`,
		},
	})

	// Both
	RunBp2BuildTestCaseSimple(t, Bp2BuildTestCase{
		Description:                        fmt.Sprintf("py2&3 %s converts to a native py_library", modType),
		ModuleTypeUnderTest:                modType,
		ModuleTypeUnderTestFactory:         factory,
		ModuleTypeUnderTestBp2BuildMutator: mutator,
		Blueprint: fmt.Sprintf(`%s {
    name: "foo",
    srcs: ["a.py"],
    version: {
        py2: {
            enabled: true,
        },
        py3: {
            enabled: true,
        },
    },

    bazel_module: { bp2build_available: true },
}`, modType),
		ExpectedBazelTargets: []string{
			// srcs_version is PY2ANDPY3 by default.
			`py_library(
    name = "foo",
    srcs = ["a.py"],
)`,
		},
	})
}

func TestPythonLibraryArchVariance(t *testing.T) {
	testPythonArchVariance(t, "python_library", "py_library",
		python.PythonLibraryFactory, python.PythonLibraryBp2Build,
		func(ctx android.RegistrationContext) {})
}

func TestPythonLibraryHostArchVariance(t *testing.T) {
	testPythonArchVariance(t, "python_library_host", "py_library",
		python.PythonLibraryHostFactory, python.PythonLibraryHostBp2Build,
		func(ctx android.RegistrationContext) {})
}

// TODO: refactor python_binary_conversion_test to use this
func testPythonArchVariance(t *testing.T, modType, bazelTarget string,
	factory android.ModuleFactory, mutator PythonLibBp2Build,
	registration func(ctx android.RegistrationContext)) {
	t.Helper()
	RunBp2BuildTestCase(t, registration, Bp2BuildTestCase{
		Description:                        fmt.Sprintf("test %s arch variants", modType),
		ModuleTypeUnderTest:                modType,
		ModuleTypeUnderTestFactory:         factory,
		ModuleTypeUnderTestBp2BuildMutator: mutator,
		Filesystem: map[string]string{
			"dir/arm.py": "",
			"dir/x86.py": "",
		},
		Blueprint: fmt.Sprintf(`%s {
					 name: "foo",
					 arch: {
						 arm: {
							 srcs: ["arm.py"],
						 },
						 x86: {
							 srcs: ["x86.py"],
						 },
					},
				 }`, modType),
		ExpectedBazelTargets: []string{
			fmt.Sprintf(`%s(
    name = "foo",
    srcs = select({
        "//build/bazel/platforms/arch:arm": ["arm.py"],
        "//build/bazel/platforms/arch:x86": ["x86.py"],
        "//conditions:default": [],
    }),
    srcs_version = "PY3",
)`, bazelTarget),
		},
	})
}
