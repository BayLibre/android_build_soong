package metrics

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRepo(t *testing.T) {
	// Setup repo directory with the default.xml file
	topDir := t.TempDir()
	manifestDir := filepath.Join(topDir, ".repo", "manifests")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatalf("failed to create manifest directory %q: %v", topDir, err)
	}

	manifestFilename := filepath.Join(manifestDir, "default.xml")
	if err := ioutil.WriteFile(manifestFilename, xmlData, 0666); err != nil {
		t.Fatalf("failed to write manifest file %q: %v", manifestFilename, err)
	}

	m := New()
	if err := m.Repo(topDir); err != nil {
		t.Fatalf("got %v, expecting nil for Repo metric", err)
	}

	expectedBranchName := "master"
	if *m.metrics.Repo.DefaultBranchName != expectedBranchName {
		t.Errorf("got %s, want %s for default branch name", *m.metrics.Repo.DefaultBranchName, expectedBranchName)
	}

	expectedRepoName := "aosp"
	if *m.metrics.Repo.DefaultRepoName != expectedRepoName {
		t.Errorf("got %s, want %s for default repo name", *m.metrics.Repo.DefaultRepoName, expectedRepoName)
	}

	expectedRepoUrl := "https://android-review.googlesource.com/"
	if *m.metrics.Repo.DefaultRepoUrl != expectedRepoUrl {
		t.Errorf("got %s, want %s for the default repo URL", *m.metrics.Repo.DefaultRepoUrl, expectedRepoUrl)
	}
}

func TestRepoErrors(t *testing.T) {
	// Setup repo directory with the default.xml file
	topDir := t.TempDir()
	manifestDir := filepath.Join(topDir, ".repo", "manifests")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatalf("failed to create manifest directory %q: %v", topDir, err)
	}

	tests := []struct {
		description        string
		createManifestFile bool
		manifestData       []byte
	}{{
		description: "manifest file does not exist",
	}, {
		description:        "corrupted manifest file",
		createManifestFile: true,
		manifestData:       []byte("data is corrupted"),
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.createManifestFile {
				manifestFilename := filepath.Join(manifestDir, "default.xml")
				if err := ioutil.WriteFile(manifestFilename, tt.manifestData, 0666); err != nil {
					t.Fatalf("failed to write manifest file %q: %v", manifestFilename, err)
				}
			}

			m := New()
			if err := m.Repo(topDir); err == nil {
				t.Error("got nil, expecting error for Repo metric")
			}

		})
	}
}

var xmlData = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<manifest>
   <remote name="aosp"
           fetch=".."
           review="https://android-review.googlesource.com/" />
   <default revision="master"
           remote="aosp"
           sync-j="4" />
</manifest>
`)
