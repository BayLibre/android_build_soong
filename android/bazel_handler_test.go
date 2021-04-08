package android

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetOutputFiles(t *testing.T) {
	label := "//foo:bar"
	arch := Arm64
	bazelContext, _ := testBazelContext(t, map[string]string{
		"cquery": `@sourceroot//foo:bar|arm64>>out/foo/bar.txt`,
	})
	g, ok := bazelContext.GetOutputFiles(label, arch)
	err := bazelContext.InvokeBazel()
	if err != nil {
		t.Errorf("Did not expect error invoking Bazel, but got %s", err)
	}
	g, ok = bazelContext.GetOutputFiles(label, arch)
	if !ok {
		t.Errorf("Expected cquery results after running InvokeBazel(), but got none")
	} else if w := []string{"out/foo/bar.txt"}; !reflect.DeepEqual(w, g) {
		t.Errorf("Expected output %s, got %s", w, g)
	}
}

func TestGetOutputFilesAndCcObjectFiles(t *testing.T) {
	label := "//foo:bar"
	arch := Arm64
	bazelContext, _ := testBazelContext(t, map[string]string{
		"cquery": `@sourceroot//foo:bar|arm64>>out/foo/bar.a|out/foo/bar.o, out/foo/baz.o`,
	})
	bazelContext.GetOutputFilesAndCcObjectFiles(label, arch)
	err := bazelContext.InvokeBazel()
	if err != nil {
		t.Errorf("Did not expect error invoking Bazel, but got %s", err)
	}
	outputs, objects, ok := bazelContext.GetOutputFilesAndCcObjectFiles(label, arch)
	if !ok {
		t.Errorf("Expected cquery results after running InvokeBazel(), but got none")
	} else {
		if w := []string{"out/foo/bar.a"}; !reflect.DeepEqual(w, outputs) {
			t.Errorf("Expected output %s, got %s", w, outputs)
		}
		if w := []string{"out/foo/bar.o", "out/foo/baz.o"}; !reflect.DeepEqual(w, objects) {
			t.Errorf("Expected cc objects %s, got %s", w, objects)
		}
	}
}

func TestRequestResultsAfterInvokeBazel(t *testing.T) {
	label := "//foo:bar"
	arch := Arm64
	bazelContext, _ := testBazelContext(t, map[string]string{
		"cquery": `@sourceroot//foo:bar|arm64>>out/foo/bar.txt`,
	})
	g, ok := bazelContext.GetOutputFiles(label, arch)
	if ok {
		t.Errorf("Did not expect cquery results prior to running InvokeBazel(), but got %s", g)
	}
	err := bazelContext.InvokeBazel()
	if err != nil {
		t.Fatalf("Did not expect error invoking Bazel, but got %s", err)
	}
	g, ok = bazelContext.GetOutputFiles(label, arch)
	if !ok {
		t.Errorf("Expected cquery results after running InvokeBazel(), but got none")
	} else if w := []string{"out/foo/bar.txt"}; !reflect.DeepEqual(w, g) {
		t.Errorf("Expected output %s, got %s", w, g)
	}
}

func TestInvokeBazelWritesBazelFiles(t *testing.T) {
	bazelContext, baseDir := testBazelContext(t, map[string]string{})
	err := bazelContext.InvokeBazel()
	if err != nil {
		t.Fatalf("Did not expect error invoking Bazel, but got %s", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "bazel", "main.bzl")); os.IsNotExist(err) {
		t.Errorf("Expected main.bzl to exist, but it does not")
	} else if err != nil {
		t.Errorf("Unexpected error stating main.bzl %s", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "bazel", "BUILD.bazel")); os.IsNotExist(err) {
		t.Errorf("Expected BUILD.bazel to exist, but it does not")
	} else if err != nil {
		t.Errorf("Unexpected error stating BUILD.bazel %s", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "bazel", "WORKSPACE.bazel")); os.IsNotExist(err) {
		t.Errorf("Expected WORKSPACE.bazel to exist, but it does not")
	} else if err != nil {
		t.Errorf("Unexpected error stating WORKSPACE.bazel %s", err)
	}
}

func TestInvokeBazelPopulatesBuildStatements(t *testing.T) {
	bazelContext, _ := testBazelContext(t, map[string]string{
		"aquery": `
{
  "artifacts": [{
    "id": 1,
    "pathFragmentId": 1
  }, {
    "id": 2,
    "pathFragmentId": 2
  }],
  "actions": [{
    "targetId": 1,
    "actionKey": "x",
    "mnemonic": "x",
    "arguments": ["touch", "foo"],
    "inputDepSetIds": [1],
    "outputIds": [1],
    "primaryOutputId": 1
  }],
  "depSetOfFiles": [{
    "id": 1,
    "directArtifactIds": [1, 2]
  }],
  "pathFragments": [{
    "id": 1,
    "label": "one"
  }, {
    "id": 2,
    "label": "two"
  }]
}`,
	})
	err := bazelContext.InvokeBazel()
	if err != nil {
		t.Fatalf("Did not expect error invoking Bazel, but got %s", err)
	}

	got := bazelContext.BuildStatementsToRegister()
	if want := 1; len(got) != want {
		t.Errorf("Expected %d registered build statements, got %#v", want, got)
	}
}

func testBazelContext(t *testing.T, bazelCommandResults map[string]string) (*bazelContext, string) {
	t.Helper()
	p := bazelPaths{
		buildDir:     t.TempDir(),
		outputBase:   "outputbase",
		workspaceDir: "workspace_dir",
	}
	if _, exists := bazelCommandResults["aquery"]; !exists {
		bazelCommandResults["aquery"] = "{}\n"
	}
	return &bazelContext{
		bazelRunner: &mockBazelRunner{bazelCommandResults: bazelCommandResults},
		paths:       &p,
		requests:    map[cqueryKey]bool{},
	}, p.buildDir
}
