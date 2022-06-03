package android

import (
	"path/filepath"
	"testing"
)

type TestBazelPathContext struct{}

func (*TestBazelPathContext) Config() Config {
	cfg := NullConfig("out", "out/soong")
	cfg.BazelContext = MockBazelContext{
		OutputBaseDir: "out/bazel",
	}
	return cfg
}

func (*TestBazelPathContext) AddNinjaFileDeps(deps ...string) {
	panic("Unimplemented")
}

func TestPathForBazelOut(t *testing.T) {
	ctx := &TestBazelPathContext{}
	out := PathForBazelOut(ctx, "foo/bar/baz/boq.txt")
	expectedPath := filepath.Join(ctx.Config().BazelContext.OutputBase(), "execroot/__main__/foo/bar/baz/boq.txt")
	if out.String() != expectedPath {
		t.Errorf("incorrect OutputPath: expected %q, got %q", expectedPath, out.String())
	}

	expectedRelPath := "foo/bar/baz/boq.txt"
	if out.Rel() != expectedRelPath {
		t.Errorf("incorrect OutputPath.Rel(): expected %q, got %q", expectedRelPath, out.Rel())
	}
}

func TestPathForBazelOutRelative(t *testing.T) {
	ctx := &TestBazelPathContext{}
	out := PathForBazelOutRelative(ctx, "foo/bar", "foo/bar/baz/boq.txt")

	expectedPath := filepath.Join(ctx.Config().BazelContext.OutputBase(), "execroot/__main__/foo/bar/baz/boq.txt")
	if out.String() != expectedPath {
		t.Errorf("incorrect OutputPath: expected %q, got %q", expectedPath, out.String())
	}

	expectedRelPath := "baz/boq.txt"
	if out.Rel() != expectedRelPath {
		t.Errorf("incorrect OutputPath.Rel(): expected %q, got %q", expectedRelPath, out.Rel())
	}
}

func TestPathForBazelOutRelativeWithParentDirectoryRoot(t *testing.T) {
	ctx := &TestBazelPathContext{}
	out := PathForBazelOutRelative(ctx, "../bazel_tools", "../bazel_tools/foo/bar/baz.sh")

	expectedPath := filepath.Join(ctx.Config().BazelContext.OutputBase(), "execroot/bazel_tools/foo/bar/baz.sh")
	if out.String() != expectedPath {
		t.Errorf("incorrect OutputPath: expected %q, got %q", expectedPath, out.String())
	}

	expectedRelPath := "foo/bar/baz.sh"
	if out.Rel() != expectedRelPath {
		t.Errorf("incorrect OutputPath.Rel(): expected %q, got %q", expectedRelPath, out.Rel())
	}
}
