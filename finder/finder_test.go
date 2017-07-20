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

	"bytes"

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
   M. Filepaths that aren't readable by the current user
2. With db, run once:
   A. Mismatched db version header
   B. Corrupt db
3. With db, run twice:
   A. No changes
   B. Files added
   C. Files deleted
   D. Directories added having files that match the search criteria
   E. Directories swapped
   F. Change of the Device attribute on a directory
   G. Confirm the bytes of the cache are consistent across runs
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
	finder := New(cacheParams, filesystem, *log.New(os.Stdout, "", 0), cachePath)
	return finder
}

func Write(t *testing.T, path string, content string, filesystem *pathtools.MockFs) {
	parent := filepath.Dir(path)
	filesystem.MkDirs(parent)
	err := filesystem.WriteFile(path, []byte(content), 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func Create(t *testing.T, path string, filesystem *pathtools.MockFs) {
	Write(t, path, "hi", filesystem)
}

func Delete(t *testing.T, path string, filesystem *pathtools.MockFs) {
	err := filesystem.Remove(path)
	if err != nil {
		t.Fatal(err.Error())
	}
}
func RemoveAll(t *testing.T, path string, filesystem *pathtools.MockFs) {
	err := filesystem.RemoveAll(path)
	if err != nil {
		t.Fatal(err.Error())
	}
}
func Move(t *testing.T, oldPath string, newPath string, filesystem *pathtools.MockFs) {
	err := filesystem.Rename(oldPath, newPath)
	if err != nil {
		t.Fatal(err.Error())
	}
}
func assertSameResponse(t *testing.T, actual []string, expected []string) {
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		message := fmt.Sprintf(
			"Expected Finder to return these %v paths:\n  %v,\ninstead returned these %v paths:  %v",
			len(expected), expected, len(actual), actual)
		t.Errorf(message)
		t.FailNow()
	}
}

func assertSameStatCalls(t *testing.T, actual []string, expected []string) {
	sort.Strings(actual)
	sort.Strings(expected)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf(
			"Finder made incorrect Stat calls.\n"+
				"Actual:\n"+
				"%v\n"+
				"Expected:\n"+
				"%v\n"+
				"\n",
			actual, expected)
	}
}
func assertSameReadDirCalls(t *testing.T, actual []string, expected []string) {
	sort.Strings(actual)
	sort.Strings(expected)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf(
			"Finder made incorrect ReadDir calls.\n"+
				"Actual:\n"+
				"%v\n"+
				"Expected:\n"+
				"%v\n"+
				"\n",
			actual, expected)
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
	defer finder.Shutdown()

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

func TestEmptyPath(t *testing.T) {
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
	finder.SetWorkingDir("/cwd")
	defer finder.Shutdown()

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
	defer finder.Shutdown()

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
	defer finder.Shutdown()

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
	defer finder.Shutdown()

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
	defer finder.Shutdown()

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
	defer finder.Shutdown()

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
	// If the caller queries for a file that is in the cache, then computing the
	// correct answer won't be fast, and it would be easy for the caller to
	// fail to notice its slowness. Instead, we only ever search the cache for files
	// to return, which enforces that we can determine which files will be
	// interesting upfront.
	assertSameResponse(t, foundPaths, []string{})

	finder.Shutdown()
}

func TestSearchingForFilesExcludedFromCache(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/a/misc.txt", filesystem)

	// set up the finder and run it
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "misc.txt")
	// If the caller queries for a file that is in the cache, then computing the
	// correct answer won't be fast, and it would be easy for the caller to
	// fail to notice its slowness. Instead, we only ever search the cache for files
	// to return, which enforces that we can determine which files will be
	// interesting upfront.
	assertSameResponse(t, foundPaths, []string{})

	finder.Shutdown()
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
			[]string{"/cwd", "/tmp"},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})
	defer finder.Shutdown()

	finder.SetWorkingDir("/cwd")

	foundPaths := finder.FindNamed("a", "findme.txt")

	assertSameResponse(t, foundPaths,
		[]string{"a/a/findme.txt",
			"a/findme.txt"})
}

// have to run this test with the race-detector (`go test -race src/android/soong/finder/*.go`)
// in order for there to be much chance of the test actually failing in practice
func TestRootDirsContainedInOtherRootDirs(t *testing.T) {
	filesystem := NewFs()

	Create(t, "/tmp/a/b/c/d/e/f/g/h/i/j/findme.txt", filesystem)

	finder := NewFinder(t,
		filesystem,
		CacheParams{
			[]string{"/", "/a/b/c", "/a/b/c/d/e/f", "/a/b/c/d/e/f/g/h/i"},
			[]string{},
			[]string{},
			[]string{"findme.txt"}})
	defer finder.Shutdown()

	foundPaths := finder.FindNamed("/tmp/a", "findme.txt")

	assertSameResponse(t, foundPaths,
		[]string{"/tmp/a/b/c/d/e/f/g/h/i/j/findme.txt"})
}

func TestConcurrentFindSameDirectory(t *testing.T) {
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
	defer finder.Shutdown()

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

func TestConcurrentFindDifferentDirectories(t *testing.T) {
	filesystem := NewFs()

	// create a bunch of files and directories
	allFiles := []string{}
	numSubdirs := 10
	rootPaths := []string{}
	queryAnswers := [][]string{}
	for i := 0; i < numSubdirs; i++ {
		parentDir := fmt.Sprintf("/tmp/%v", i)
		rootPaths = append(rootPaths, parentDir)
		queryAnswers = append(queryAnswers, []string{})
		for j := 0; j < 10; j++ {
			filePath := filepath.Join(parentDir, fmt.Sprintf("%v/findme.txt", j))
			queryAnswers[i] = append(queryAnswers[i], filePath)
			allFiles = append(allFiles, filePath)
		}
		sort.Strings(queryAnswers[i])
	}
	sort.Strings(allFiles)
	for _, path := range allFiles {
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
	defer finder.Shutdown()

	type testRun struct {
		path           string
		foundMatches   []string
		correctMatches []string
	}

	numTests := numSubdirs + 1
	testRuns := make(chan testRun, numTests)

	searchAt := func(path string, correctMatches []string) {
		foundPaths := finder.FindNamed(path, "findme.txt")
		testRuns <- testRun{path, foundPaths, correctMatches}
	}

	// make several parallel calls to the finder
	go searchAt("/tmp", allFiles)
	for i := 0; i < len(rootPaths); i++ {
		go searchAt(rootPaths[i], queryAnswers[i])
	}

	// check that each response was correct
	for i := 0; i < numTests; i++ {
		testRun := <-testRuns
		assertSameResponse(t, testRun.foundMatches, testRun.correctMatches)
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
	defer finder.Shutdown()

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
	defer finder.Shutdown()

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

	// check results
	reader, err := filesystem.Open(finder.DbPath)
	if err != nil {
		t.Fatalf("failed to save cache db, %v", err)
	}
	cacheBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to save cache db, %v", err)
	}
	cacheText := string(cacheBytes)
	if len(cacheText) < 1 {
		if err != nil {
			t.Fatalf("saved cache db is empty\n")
		}
	}
	if len(filesystem.StatCalls) == 0 {
		t.Fatal("No Stat calls recorded by mock filesystem")
	}
	if len(filesystem.ReadDirCalls) == 0 {
		t.Fatal("No ReadDir calls recorded by filesystem")
	}
	statCalls := filesystem.StatCalls
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")
	// check results
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{})
	assertSameReadDirCalls(t, filesystem.StatCalls, statCalls)

	finder2.Shutdown()
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
	finder.Shutdown()

	// check the response of the first finder
	correctResponse := []string{"/tmp/a/findme.txt",
		"/tmp/findme.txt"}
	assertSameResponse(t, foundPaths, correctResponse)
	numStatCalls := len(filesystem.StatCalls)
	numReadDirCalls := len(filesystem.ReadDirCalls)

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
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")
	// check results
	assertSameResponse(t, foundPaths, correctResponse)
	numNewStatCalls := len(filesystem.StatCalls)
	numNewReadDirCalls := len(filesystem.ReadDirCalls)
	// It's permissable to make more Stat calls with a corrupted cache because
	// the Finder may restart once it detects corruption.
	// However, it may have already issued many Stat calls.
	// Because a corrupted db is not expected to be a common (or even a supported case),
	// we don't care to optimize it and don't cache the already-issued Stat calls
	if numNewReadDirCalls < numReadDirCalls {
		t.Fatalf(
			"Finder made fewer ReadDir calls with a corrupted cache (%v calls) than with no cache"+
				" (%v calls)",
			numNewReadDirCalls, numReadDirCalls)
	}
	if numNewStatCalls < numStatCalls {
		t.Fatalf(
			"Finder made fewer Stat calls with a corrupted cache (%v calls) than with no cache (%v calls)",
			numNewStatCalls, numStatCalls)
	}
	finder2.Shutdown()
}

func TestStatCalls(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/a/findme.txt", filesystem)

	// run finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()

	// check response
	assertSameResponse(t, foundPaths, []string{"/tmp/a/findme.txt"})
	assertSameStatCalls(t, filesystem.StatCalls, []string{"/tmp", "/tmp/a"})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{"/tmp", "/tmp/a"})
}

func TestFileAdded(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/ignoreme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/b/ignore.txt", filesystem)
	Create(t, "/tmp/b/c/nope.txt", filesystem)
	Create(t, "/tmp/b/c/d/irrelevant.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()
	// check the response of the first finder
	assertSameResponse(t, foundPaths, []string{"/tmp/a/findme.txt"})

	// modify the filesystem
	filesystem.Clock.Tick()
	Create(t, "/tmp/b/c/findme.txt", filesystem)
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")

	// check results
	assertSameResponse(t, foundPaths, []string{"/tmp/a/findme.txt", "/tmp/b/c/findme.txt"})
	assertSameStatCalls(t, filesystem.StatCalls, []string{"/tmp", "/tmp/a", "/tmp/b", "/tmp/b/c", "/tmp/b/c/d"})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{"/tmp/b/c"})

	finder2.Shutdown()
}

func TestDirectoriesAdded(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/ignoreme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/b/ignore.txt", filesystem)
	Create(t, "/tmp/b/c/nope.txt", filesystem)
	Create(t, "/tmp/b/c/d/irrelevant.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()
	// check the response of the first finder
	assertSameResponse(t, foundPaths, []string{"/tmp/a/findme.txt"})

	// modify the filesystem
	filesystem.Clock.Tick()
	Create(t, "/tmp/b/c/new/findme.txt", filesystem)
	Create(t, "/tmp/b/c/new/new2/findme.txt", filesystem)
	Create(t, "/tmp/b/c/new/new2/ignoreme.txt", filesystem)
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")

	// check results
	assertSameResponse(t, foundPaths,
		[]string{"/tmp/a/findme.txt", "/tmp/b/c/new/findme.txt", "/tmp/b/c/new/new2/findme.txt"})
	assertSameStatCalls(t, filesystem.StatCalls,
		[]string{"/tmp", "/tmp/a", "/tmp/b", "/tmp/b/c", "/tmp/b/c/d", "/tmp/b/c/new", "/tmp/b/c/new/new2"})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{"/tmp/b/c", "/tmp/b/c/new", "/tmp/b/c/new/new2"})

	finder2.Shutdown()
}

func TestFileDeleted(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/ignoreme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/b/findme.txt", filesystem)
	Create(t, "/tmp/b/c/nope.txt", filesystem)
	Create(t, "/tmp/b/c/d/irrelevant.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()
	// check the response of the first finder
	assertSameResponse(t, foundPaths, []string{"/tmp/a/findme.txt", "/tmp/b/findme.txt"})

	// modify the filesystem
	filesystem.Clock.Tick()
	Delete(t, "/tmp/b/findme.txt", filesystem)
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")

	// check results
	assertSameResponse(t, foundPaths, []string{"/tmp/a/findme.txt"})
	assertSameStatCalls(t, filesystem.StatCalls, []string{"/tmp", "/tmp/a", "/tmp/b", "/tmp/b/c", "/tmp/b/c/d"})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{"/tmp/b"})

	finder2.Shutdown()
}

func TestDirectoriesDeleted(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/a/1/findme.txt", filesystem)
	Create(t, "/tmp/a/1/2/findme.txt", filesystem)
	Create(t, "/tmp/b/findme.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()
	// check the response of the first finder
	assertSameResponse(t, foundPaths,
		[]string{"/tmp/findme.txt",
			"/tmp/a/findme.txt",
			"/tmp/a/1/findme.txt",
			"/tmp/a/1/2/findme.txt",
			"/tmp/b/findme.txt"})

	// modify the filesystem
	filesystem.Clock.Tick()
	RemoveAll(t, "/tmp/a/1", filesystem)
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")

	// check results
	assertSameResponse(t, foundPaths,
		[]string{"/tmp/findme.txt", "/tmp/a/findme.txt", "/tmp/b/findme.txt"})
	// Technically, we don't care whether /tmp/a/1/2 gets Statted or gets skipped
	// if the Finder detects the nonexistence of /tmp/a/1
	// However, when resuming from cache, we don't want the Finder to necessarily wait
	// to stat a directory until after statting its parent.
	// So here we just include /tmp/a/1/2 in the list.
	// The Finder is currently implemented to always restat every dir and
	// to not short-circuit due to nonexistence of parents (but it will remove
	// missing dirs from the cache for next time)
	assertSameStatCalls(t, filesystem.StatCalls,
		[]string{"/tmp", "/tmp/a", "/tmp/a/1", "/tmp/a/1/2", "/tmp/b"})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{"/tmp/a"})

	finder2.Shutdown()
}

func TestDirectoriesMoved(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/a/1/findme.txt", filesystem)
	Create(t, "/tmp/a/1/2/findme.txt", filesystem)
	Create(t, "/tmp/b/findme.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()
	// check the response of the first finder
	assertSameResponse(t, foundPaths,
		[]string{"/tmp/findme.txt",
			"/tmp/a/findme.txt",
			"/tmp/a/1/findme.txt",
			"/tmp/a/1/2/findme.txt",
			"/tmp/b/findme.txt"})

	// modify the filesystem
	filesystem.Clock.Tick()
	Move(t, "/tmp/a", "/tmp/c", filesystem)
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")

	// check results
	assertSameResponse(t, foundPaths,
		[]string{"/tmp/findme.txt",
			"/tmp/b/findme.txt",
			"/tmp/c/findme.txt",
			"/tmp/c/1/findme.txt",
			"/tmp/c/1/2/findme.txt"})
	// Technically, we don't care whether /tmp/a/1/2 gets Statted or gets skipped
	// if the Finder detects the nonexistence of /tmp/a/1
	// However, when resuming from cache, we don't want the Finder to necessarily wait
	// to stat a directory until after statting its parent.
	// So here we just include /tmp/a/1/2 in the list.
	// The Finder is currently implemented to always restat every dir and
	// to not short-circuit due to nonexistence of parents (but it will remove
	// missing dirs from the cache for next time)
	assertSameStatCalls(t, filesystem.StatCalls,
		[]string{"/tmp", "/tmp/a", "/tmp/a/1", "/tmp/a/1/2", "/tmp/b", "/tmp/c", "/tmp/c/1", "/tmp/c/1/2"})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{"/tmp", "/tmp/c", "/tmp/c/1", "/tmp/c/1/2"})
	finder2.Shutdown()
}

func TestChangeOfDevice(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/subdir/findme.txt", filesystem)

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()
	// check the response of the first finder
	assertSameResponse(t, foundPaths,
		[]string{"/tmp/findme.txt", "/tmp/subdir/findme.txt"})

	// replace the Device with another one
	filesystem.SetId(filesystem.Id() + "_changed")
	filesystem.ClearMetrics()

	// run the second finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	foundPaths = finder2.FindNamed("/tmp", "findme.txt")

	// check results
	assertSameResponse(t, foundPaths,
		[]string{"/tmp/findme.txt", "/tmp/subdir/findme.txt"})
	assertSameStatCalls(t, filesystem.StatCalls,
		[]string{"/tmp", "/tmp/subdir"})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{"/tmp", "/tmp/subdir"})
	finder2.Shutdown()
}

func TestConsistentCacheOrdering(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	for i := 0; i < 5; i++ {
		Create(t, fmt.Sprintf("/tmp/%v/findme.txt", i), filesystem)
	}

	// run the first finder
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	finder.FindNamed("/tmp", "findme.txt")
	finder.Shutdown()

	// read db file
	reader1, err := filesystem.Open(finder.DbPath)
	if err != nil {
		t.Fatal(err)
	}
	data1, err := ioutil.ReadAll(reader1)
	if err != nil {
		t.Fatal(err)
	}

	err = filesystem.Remove(finder.DbPath)
	if err != nil {
		t.Fatal(err)
	}

	// run another finder
	finder2 := NewFinder(t, filesystem, cacheParams)
	finder2.FindNamed("/tmp", "findme.txt")
	finder2.Shutdown()

	reader2, err := filesystem.Open(finder.DbPath)
	if err != nil {
		t.Fatal(err)
	}
	data2, err := ioutil.ReadAll(reader2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data1, data2) {
		string1 := string(data1)
		string2 := string(data2)
		t.Errorf("Running Finder twice generated two dbs not having identical contents.\n"+
			"Content of first file:\n"+
			"\n"+
			"%v"+
			"\n"+
			"\n"+
			"Content of second file:\n"+
			"\n"+
			"%v\n"+
			"\n",
			string1,
			string2,
		)
	}

}

func TestNumSyscallsOfSecondFind(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/a/misc.txt", filesystem)

	// set up the finder and run it once
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	assertSameResponse(t, foundPaths, []string{"/tmp/findme.txt", "/tmp/a/findme.txt"})

	filesystem.ClearMetrics()

	// run the finder again and confirm it doesn't check the filesystem
	refoundPaths := finder.FindNamed("/tmp", "findme.txt")
	assertSameResponse(t, refoundPaths, foundPaths)

	assertSameStatCalls(t, filesystem.StatCalls, []string{})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{})

	finder.Shutdown()
}

func TestChangingParamsOfSecondFind(t *testing.T) {
	// setup filesystem
	filesystem := NewFs()
	Create(t, "/tmp/findme.txt", filesystem)
	Create(t, "/tmp/a/findme.txt", filesystem)
	Create(t, "/tmp/a/metoo.txt", filesystem)

	// set up the finder and run it once
	cacheParams := CacheParams{
		[]string{"/tmp"},
		[]string{},
		[]string{},
		[]string{"findme.txt", "metoo.txt"}}
	finder := NewFinder(t, filesystem, cacheParams)
	foundPaths := finder.FindNamed("/tmp", "findme.txt")
	assertSameResponse(t, foundPaths, []string{"/tmp/findme.txt", "/tmp/a/findme.txt"})

	filesystem.ClearMetrics()

	// run the finder again and confirm it doesn't check the filesystem
	refoundPaths := finder.FindNamed("/tmp", "metoo.txt")
	assertSameResponse(t, refoundPaths, []string{"/tmp/a/metoo.txt"})

	assertSameStatCalls(t, filesystem.StatCalls, []string{})
	assertSameReadDirCalls(t, filesystem.ReadDirCalls, []string{})

	finder.Shutdown()
}
