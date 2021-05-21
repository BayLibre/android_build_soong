package bp2build

import (
	"android/soong/android"
	"android/soong/python"
	"fmt"
	"strings"
	"testing"
)

type testCase struct {
	description                        string
	moduleTypeUnderTest                string
	moduleTypeUnderTestFactory         android.ModuleFactory
	moduleTypeUnderTestBp2BuildMutator func(android.TopDownMutatorContext)
	blueprint                          string
	expectedBazelTargets               []string
	filesystem                         map[string]string
}

func runTestCase(t *testing.T, tc testCase) {
	dir := "."
	filesystem := make(map[string][]byte)
	toParse := []string{
		"Android.bp",
	}
	for f, content := range tc.filesystem {
		if strings.HasSuffix(f, "Android.bp") {
			toParse = append(toParse, f)
		}
		filesystem[f] = []byte(content)
	}
	config := android.TestConfig(buildDir, nil, tc.blueprint, filesystem)
	ctx := android.NewTestContext(config)

	ctx.RegisterModuleType(tc.moduleTypeUnderTest, tc.moduleTypeUnderTestFactory)
	ctx.RegisterBp2BuildMutator(tc.moduleTypeUnderTest, tc.moduleTypeUnderTestBp2BuildMutator)
	ctx.RegisterForBazelConversion()

	_, errs := ctx.ParseFileList(dir, toParse)
	if errored(t, tc.description, errs) {
		return
	}
	_, errs = ctx.ResolveDependencies(config)
	if errored(t, tc.description, errs) {
		return
	}

	codegenCtx := NewCodegenContext(config, *ctx.Context, Bp2Build)
	bazelTargets := generateBazelTargetsForDir(codegenCtx, dir)
	if actualCount, expectedCount := len(bazelTargets), len(tc.expectedBazelTargets); actualCount != expectedCount {
		fmt.Println(bazelTargets)
		t.Errorf("%s: Expected %d bazel target, got %d", tc.description, expectedCount, actualCount)
	} else {
		for i, target := range bazelTargets {
			if w, g := tc.expectedBazelTargets[i], target.content; w != g {
				t.Errorf(
					"%s: Expected generated Bazel target to be '%s', got '%s'",
					tc.description,
					w,
					g,
				)
			}
		}
	}
}

func TestPythonBinaryHostSimple(t *testing.T) {
	runTestCase(t, testCase{
		description:                        "simple python_binary_host converts to a native py_binary",
		moduleTypeUnderTest:                "python_binary_host",
		moduleTypeUnderTestFactory:         python.PythonBinaryHostFactory,
		moduleTypeUnderTestBp2BuildMutator: python.PythonBinaryBp2Build,
		filesystem: map[string]string{
			"a.py":           "",
			"b/c.py":         "",
			"b/d.py":         "",
			"b/e.py":         "",
			"files/data.txt": "",
		},
		blueprint: `python_binary_host {
    name: "foo",
    main: "a.py",
    srcs: ["**/*.py"],
    exclude_srcs: ["b/e.py"],
    data: ["files/data.txt",],
    bazel_module: { bp2build_available: true },
}
`,
		expectedBazelTargets: []string{`py_binary(
    name = "foo",
    data = ["files/data.txt"],
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
	runTestCase(t, testCase{
		description:                        "py2 python_binary_host",
		moduleTypeUnderTest:                "python_binary_host",
		moduleTypeUnderTestFactory:         python.PythonBinaryHostFactory,
		moduleTypeUnderTestBp2BuildMutator: python.PythonBinaryBp2Build,
		blueprint: `python_binary_host {
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
		expectedBazelTargets: []string{`py_binary(
    name = "foo",
    python_version = "PY2",
    srcs = ["a.py"],
)`,
		},
	})
}

func TestPythonBinaryHostPy3(t *testing.T) {
	runTestCase(t, testCase{
		description:                        "py3 python_binary_host",
		moduleTypeUnderTest:                "python_binary_host",
		moduleTypeUnderTestFactory:         python.PythonBinaryHostFactory,
		moduleTypeUnderTestBp2BuildMutator: python.PythonBinaryBp2Build,
		blueprint: `python_binary_host {
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
		expectedBazelTargets: []string{
			// python_version is PY3 by default.
			`py_binary(
    name = "foo",
    srcs = ["a.py"],
)`,
		},
	})
}
