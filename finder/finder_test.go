// Copyright 2017 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package finder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/blueprint/pathtools"
)

/*
Cases to test:

1. Several normal finds with no db
   A. One directory having a few files
   B. Nested directories, each having some files
   C. Directory with nothing in it
   D. Malformed path ("")
   E. Nonexistent path
   F. Different config params
      1. Exclude dirs
      2. Include files
      3. Prune files
      4. Root dirs
      5. A Find that references files not included in the Include files
      6. A relative file path
   G. Symlinks pointing to directories
   H. Symlinks pointing to files
   I. Broken symlinks
   J. Filesystem loops via symlinks
   K. Two callers of the same Finder in parallel
2. With db, run once:
   A. Mismatched db version header
   B. Corrupt db
3. With db, run twice:
   A. No changes
   B. Files added
   C. Files deleted
   D. Files renamed
   E. Directories added having matching files
   F. Directories swapped
4. With mocks:
   A. Count number of Stat and ReadDir calls with mocked filesystem and an up-to-date cache
   B. Count number of Stat and ReadDir calls with mocked filesystem and a mostly up-to-date cache
   C. Count number of Stat and ReadDir calls with mocked filesystem and no prior cache
   D. Call FindNamed twice and ensure no Stat is called for the second call
5. Misc:
   A. Run with race detector?
   B. Benchmark (timing) test?
*/
func NewFs() *pathtools.MockFs {
	return pathtools.NewMockFs(map[string][]byte{})
}
func NewFinder(t *testing.T, filesystem *pathtools.MockFs, cacheParams CacheParams) *Finder {
	finder, err := New(cacheParams, filesystem, *log.New(os.Stdout, "", 0), "/finder/finder-db")
	if err != nil {
		t.Fatal(err)
	}
	return finder
}

func Write(t *testing.T, path string, content string, filesystem *pathtools.MockFs) {
	parent := filepath.Dir(path)
	filesystem.MkDirs(parent)
	err := filesystem.WriteFile(path, []byte(content), 0777)
	if err != nil {
		t.Fatal(err)
	}
}

func Create(t *testing.T, path string, filesystem *pathtools.MockFs) {
	Write(t, path, "hi", filesystem)
}

func assertEqual(t *testing.T, actual []string, expected []string) {
	if !reflect.DeepEqual(actual, expected) {
		message := fmt.Sprintf("Expected:\n  %v,\ngot:  %v", expected, actual)
		t.Fatal(message)
	}

}

// runSimpleTests creates a few files, searches for findme.txt, and checks for the expected matches
func runSimpleTest(t *testing.T, existentPaths []string, expectedMatches []string) {
	filesystem := NewFs()
	root := "/tmp"
	filesystem.MkDirs(root)
	for _, path := range existentPaths {
		Create(t, filepath.Join(root, path), filesystem)
	}

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{root},
			nil,
			nil,
			[]string{"findme.txt", "skipme.txt"}})

	foundPaths := finder.FindNamed(root, "findme.txt")
	absoluteMatches := []string{}
	for i := range expectedMatches {
		absoluteMatches = append(absoluteMatches, filepath.Join(root, expectedMatches[i]))
	}
	assertEqual(t, foundPaths, absoluteMatches)
}

func TestSingleFile(t *testing.T) {
	runSimpleTest(t,
		[]string{"findme.txt"},
		[]string{"findme.txt"},
	)
}

func TestIncludeFiles(t *testing.T) {
	runSimpleTest(t,
		[]string{"findme.txt", "skipme.txt"},
		[]string{"findme.txt"},
	)
}

func TestNestedDirectories(t *testing.T) {
	runSimpleTest(t,
		[]string{"findme.txt", "skipme.txt", "subdir/findme.txt", "subdir/skipme.txt"},
		[]string{"findme.txt", "subdir/findme.txt"},
	)
}

func TestEmptyDirectory(t *testing.T) {
	runSimpleTest(t,
		[]string{},
		[]string{},
	)
}

func TestMalformedPath(t *testing.T) {
	filesystem := NewFs()
	root := "/tmp"
	Create(t, filepath.Join(root, "findme.txt"), filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{root},
			nil,
			nil,
			[]string{"findme.txt", "skipme.txt"}})

	foundPaths := finder.FindNamed("", "findme.txt")

	assertEqual(t, foundPaths, []string{})
}

func TestFilesystemRoot(t *testing.T) {
	filesystem := NewFs()
	root := "/"
	createdPath := "/findme.txt"
	Create(t, createdPath, filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{root},
			nil,
			nil,
			[]string{"findme.txt", "skipme.txt"}})

	foundPaths := finder.FindNamed(root, "findme.txt")

	assertEqual(t, foundPaths, []string{createdPath})
}

func TestNonexistentPath(t *testing.T) {
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/tmp/IDontExist"},
			nil,
			nil,
			[]string{"findme.txt", "skipme.txt"}})

	foundPaths := finder.FindNamed("/tmp/IAlsoDontExist", "findme.txt")

	assertEqual(t, foundPaths, []string{})
}

func TestExcludeDirs(t *testing.T) {
	filesystem := NewFs()
	Create(t, "/tmp/exclude/findme.txt", filesystem)
	Create(t, "/tmp/exclude/subdir/findme.txt", filesystem)
	Create(t, "/tmp/subdir/exclude/findme.txt", filesystem)
	Create(t, "/tmp/subdir/subdir/findme.txt", filesystem)
	Create(t, "/tmp/subdir/findme.txt", filesystem)
	Create(t, "/tmp/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/tmp"},
			[]string{"exclude"},
			nil,
			[]string{"findme.txt", "skipme.txt"}})

	foundPaths := finder.FindNamed("/tmp", "findme.txt")

	assertEqual(t, foundPaths,
		[]string{"/tmp/findme.txt",
			"/tmp/subdir/findme.txt",
			"/tmp/subdir/subdir/findme.txt"})
}

func TestPruneFiles(t *testing.T) {
	filesystem := NewFs()
	Create(t, "/tmp/out/findme.txt", filesystem)
	Create(t, "/tmp/out/.ignore-out-dir", filesystem)
	Create(t, "/tmp/out/child/findme.txt", filesystem)

	Create(t, "/tmp/out2/.ignore-out-dir", filesystem)
	Create(t, "/tmp/out2/sub/findme.txt", filesystem)

	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/include/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/tmp"},
			[]string{},
			[]string{".ignore-out-dir"},
			[]string{"findme.txt"}})

	foundPaths := finder.FindNamed("/tmp", "findme.txt")

	assertEqual(t, foundPaths,
		[]string{"/tmp/findme.txt",
			"/tmp/include/findme.txt"})
}

func TestRootDir(t *testing.T) {
	filesystem := NewFs()
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/a/subdir/findme.txt", filesystem)
	Create(t, "/tmp/b/findme.txt", filesystem)
	Create(t, "/tmp/b/subdir/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/tmp/a"},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})

	foundPaths := finder.FindNamed("/tmp/a", "findme.txt")

	assertEqual(t, foundPaths,
		[]string{"/tmp/a/findme.txt",
			"/tmp/a/subdir/findme.txt"})
}

func TestUncachedDir(t *testing.T) {
	filesystem := NewFs()
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/a/subdir/findme.txt", filesystem)
	Create(t, "/tmp/b/findme.txt", filesystem)
	Create(t, "/tmp/b/subdir/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/IDoNotExist"},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})

	foundPaths := finder.FindNamed("/tmp/a", "findme.txt")

	assertEqual(t, foundPaths,
		[]string{"/tmp/a/findme.txt",
			"/tmp/a/subdir/findme.txt"})
}

// TODO get this test working and then finish writing the rest of the tests
//func TestRelativeFilePath(t *testing.T) {
//	filesystem := NewFs()
//	Create(t, "/tmp/findme.txt", filesystem)
//	Create(t, "/tmp/a/findme.txt", filesystem)
//	Create(t, "/cwd/findme.txt", filesystem)
//	Create(t, "/cwd/a/findme.txt", filesystem)
//	Create(t, "/cwd/a/a/findme.txt", filesystem)
//
//	finder := NewFinder(t,
//		filesystem,
//		CacheParams{
//			[]string{"/tmp/a"},
//			[]string{},
//			[]string{},
//			[]string{"findme.txt"}})
//
//	foundPaths := finder.FindNamed("a", "findme.txt")
//
//	assertEqual(t, foundPaths,
//		[]string{"a/findme.txt",
//			"a/subdir/findme.txt"})
//}
