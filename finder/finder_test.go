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

	"sort"

	"io/ioutil"

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
   L. Filepaths that haven't gone through filepath.Clean yet
2. With db, run once:
   A. Mismatched db version header
   B. Corrupt db
3. With db, run twice:
   A. No changes
   B. Files added
   C. Files deleted
   D. Files renamed
   E. Directories added having files that match the search criteria
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
	cachePath := "/finder/finder-db"
	cacheDir := filepath.Dir(cachePath)
	filesystem.MkDirs(cacheDir)
	finder, err := New(cacheParams, filesystem, *log.New(os.Stdout, "", 0), cachePath)
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

func assertSameResponse(t *testing.T, actual []string, expected []string) {
	if !reflect.DeepEqual(actual, expected) {
		message := fmt.Sprintf(
			"Expected Finder to return these %v paths:\n  %v,\ninstead returned these %v paths:  %v",
			len(expected), expected, len(actual), actual)
		t.Errorf(message)
		t.FailNow()
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
	assertSameResponse(t, foundPaths, absoluteMatches)
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

	assertSameResponse(t, foundPaths, []string{})
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

	assertSameResponse(t, foundPaths, []string{createdPath})
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

	assertSameResponse(t, foundPaths, []string{})
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

	assertSameResponse(t, foundPaths,
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

	assertSameResponse(t, foundPaths,
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

	assertSameResponse(t, foundPaths,
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

	assertSameResponse(t, foundPaths,
		[]string{"/tmp/a/findme.txt",
			"/tmp/a/subdir/findme.txt"})
}

func TestRelativeFilePath(t *testing.T) {
	filesystem := NewFs()

	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/b/findme.txt", filesystem)
	Create(t, "/cwd/findme.txt", filesystem)
	Create(t, "/cwd/a/findme.txt", filesystem)
	Create(t, "/cwd/a/a/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/tmp"},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})

	finder.SetWorkingDir("/cwd")

	foundPaths := finder.FindNamed("a", "findme.txt")

	assertSameResponse(t, foundPaths,
		[]string{"a/a/findme.txt",
			"a/findme.txt"})
}

func TestConcurrentFind(t *testing.T) {
	filesystem := NewFs()

	// create a bunch of files and directories
	paths := []string{}
	for i := 0; i < 10; i++ {
		parentDir := fmt.Sprintf("/tmp/%v", i)
		for j := 0; j < 10; j++ {
			filePath := filepath.Join(parentDir, fmt.Sprintf("%v/findme.txt", j))
			paths = append(paths, filePath)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		Create(t, path, filesystem)
	}

	// set up a finder
	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/tmp"},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})

	numTests := 20
	results := make(chan []string, numTests)
	// make several parallel calls to the finder
	for i := 0; i < numTests; i++ {
		go func() {
			foundPaths := finder.FindNamed("/tmp", "findme.txt")
			results <- foundPaths
		}()
	}

	// check that each response was correct
	for i := 0; i < numTests; i++ {
		foundPaths := <-results
		assertSameResponse(t, foundPaths, paths)
	}
}

func TestStrangelyFormattedPaths(t *testing.T) {
	filesystem := NewFs()

	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/b/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"//tmp//a//.."},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})

	foundPaths := finder.FindNamed("//tmp//a//..", "findme.txt")

	assertSameResponse(t, foundPaths,
		[]string{"/tmp/a/findme.txt",
			"/tmp/b/findme.txt",
			"/tmp/findme.txt"})
}

func TestCorruptedCacheHeader(t *testing.T) {
	filesystem := NewFs()

	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Write(t, "/finder/finder-db", "sample header", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/tmp"},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})

	foundPaths := finder.FindNamed("/tmp", "findme.txt")

	assertSameResponse(t, foundPaths,
		[]string{"/tmp/a/findme.txt",
			"/tmp/findme.txt"})
}

func TestCanUseCache(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	// check the response of the first finder
	correctResponse := []string{"/tmp/a/findme.txt",
		"/tmp/findme.txt"}
	assertSameResponse(t, foundPaths, correctResponse)
	finder.Shutdown()
	if filesystem.NumStatCalls == 0 {
		t.Fatal("Mock filesystem is not recording Stat calls")
	}
	if filesystem.NumReadDirCalls == 0 {
		t.Fatal("Mock filesystem is not recording ReadDir calls")
	}
	numStatCalls := filesystem.NumStatCalls
	numReadDirCalls := filesystem.NumReadDirCalls

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")
	numNewStatCalls := filesystem.NumStatCalls - numStatCalls
	numNewReadDirCalls := filesystem.NumReadDirCalls - numReadDirCalls
	if numNewReadDirCalls != 0 {
		t.Fatalf(
			"Finder made %v ReadDir calls (expected 0) with an up-to-date cache",
			numNewReadDirCalls)
	}
	if numNewStatCalls != numStatCalls {
		t.Fatalf(
			"Finder made a different number of Stat calls with an up-to-date cache (%v calls) than with no cache (%v calls)",
			numNewStatCalls, numStatCalls)
	}
}

func TestCorruptedCache(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	// check the response of the first finder
	correctResponse := []string{"/tmp/a/findme.txt",
		"/tmp/findme.txt"}
	assertSameResponse(t, foundPaths, correctResponse)
	finder.Shutdown()
	if filesystem.NumStatCalls == 0 {
		t.Fatal("Mock filesystem is not recording Stat calls")
	}
	if filesystem.NumReadDirCalls == 0 {
		t.Fatal("Mock filesystem is not recording ReadDir calls")
	}
	numStatCalls := filesystem.NumStatCalls
	numReadDirCalls := filesystem.NumReadDirCalls

	// load the cache file, corrupt it, and save it
	cacheReader, err := filesystem.Open("/finder/finder-db")
	if err != nil {
		t.Fatal(err)
	}
	cacheData, err := ioutil.ReadAll(cacheReader)
	if err != nil {
		t.Fatal(err)
	}
	cacheData = append(cacheData, []byte("DontMindMe")...)
	filesystem.WriteFile("/finder/finder-db", cacheData, 0777)

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")
	numNewStatCalls := filesystem.NumStatCalls - numStatCalls
	numNewReadDirCalls := filesystem.NumReadDirCalls - numReadDirCalls
	if numNewReadDirCalls < numReadDirCalls {
		t.Fatalf(
			"Finder made fewer ReadDir calls with a corrupted cache (%v calls) than with no cache (%v calls)",
			numNewReadDirCalls, numReadDirCalls)
	}
	if numNewStatCalls < numStatCalls {
		t.Fatalf(
			"Finder made fewer Stat calls with a corrupted cache (%v calls) than with no cache (%v calls)",
			numNewStatCalls, numStatCalls)
	}
}

// TODO: write more tests
//func TestFilesAdded(t *Testing.T) {
//
//}
