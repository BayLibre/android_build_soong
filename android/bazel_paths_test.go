package android

import (
	"path/filepath"
	"testing"
)

func TestPathForBazelOutStartsWithBazelOutputBasePath(t *testing.T) {
	bazelOutputBaseDir := filepath.Join("out", "bazel")
	result := GroupFixturePreparers(
		FixtureModifyConfig(func(config Config) {
			config.BazelContext = MockBazelContext{
				OutputBaseDir: bazelOutputBaseDir,
			}
		}),
	).RunTest(t)

	ctx := &TestPathContext{TestResult: result}
	paths := []string{
		"foo/bar/baz/boq.txt",
	}
	out := PathForBazelOut(ctx, paths...)
	expectedPath := filepath.Join(append(BazelOutputBasePath(ctx), "foo/bar/baz/boq.txt")...)
	if out.String() != expectedPath {
		t.Errorf("incorrect OutputPath: expected %q, got %q", expectedPath, out.String())
	}
}

func TestPathForBazelOutRelativeDoesNotIncludeBazelOutputBasePath(t *testing.T) {
	bazelOutputBaseDir := filepath.Join("out", "bazel")
	result := GroupFixturePreparers(
		FixtureModifyConfig(func(config Config) {
			config.BazelContext = MockBazelContext{
				OutputBaseDir: bazelOutputBaseDir,
			}
		}),
	).RunTest(t)

	ctx := &TestPathContext{TestResult: result}
	paths := []string{
		"baz/boq.txt",
	}
	relativeRoot := []string{
		"foo",
		"bar",
	}
	out := PathForBazelOutRelative(ctx, relativeRoot, paths...)
	expectedRel := "baz/boq.txt"
	if out.Rel() != expectedRel {
		t.Errorf("incorrect OutputPath: expected Rel() %q, got %q", expectedRel, out.String())
	}
}

func TestPathForBazelOutNotEmptyRel(t *testing.T) {
	bazelOutputBaseDir := filepath.Join("out", "bazel")
	result := GroupFixturePreparers(
		FixtureModifyConfig(func(config Config) {
			config.BazelContext = MockBazelContext{
				OutputBaseDir: bazelOutputBaseDir,
			}
		}),
	).RunTest(t)

	ctx := &TestPathContext{TestResult: result}
	paths := []string{
		"foo/bar/baz/boq.txt",
	}
	out := PathForBazelOut(ctx, paths...)
	expectedRel := "foo/bar/baz/boq.txt"
	if out.Rel() != expectedRel {
		t.Errorf("incorrect OutputPath: expected Rel() == %q, got %q", expectedRel, out.Rel())
	}
}

func TestPathForBazelOutWithRelativePathChange(t *testing.T) {
	bazelOutputBaseDir := filepath.Join("out", "bazel")
	result := GroupFixturePreparers(
		FixtureModifyConfig(func(config Config) {
			config.BazelContext = MockBazelContext{
				OutputBaseDir: bazelOutputBaseDir,
			}
		}),
	).RunTest(t)

	ctx := &TestPathContext{TestResult: result}
	paths := []string{
		"../bazel_tools/foo/bar/baz.sh",
	}
	out := PathForBazelOut(ctx, paths...)
	expectedPath := filepath.Join(bazelOutputBaseDir, "execroot/bazel_tools/foo/bar/baz.sh")
	if out.String() != expectedPath {
		t.Errorf("incorrect OutputPath: expected %q, got %q", expectedPath, out.String())
	}
}

func TestPathForBazelOutRelativeWithRelativePathChange(t *testing.T) {
	bazelOutputBaseDir := filepath.Join("out", "bazel")
	result := GroupFixturePreparers(
		FixtureModifyConfig(func(config Config) {
			config.BazelContext = MockBazelContext{
				OutputBaseDir: bazelOutputBaseDir,
			}
		}),
	).RunTest(t)

	ctx := &TestPathContext{TestResult: result}
	paths := []string{
		"foo/bar/baz.sh",
	}
	relativeRoot := []string{
		"../bazel_tools",
	}
	out := PathForBazelOutRelative(ctx, relativeRoot, paths...)
	expectedPath := filepath.Join(bazelOutputBaseDir, "execroot/bazel_tools/foo/bar/baz.sh")
	if out.String() != expectedPath {
		t.Errorf("incorrect OutputPath: expected %q, got %q", expectedPath, out.String())
	}
	expectedRel := "foo/bar/baz.sh"
	if out.Rel() != expectedRel {
		t.Errorf("incorrect OutputPath: expected Rel() == %q, got %q", expectedRel, out.Rel())
	}
}
