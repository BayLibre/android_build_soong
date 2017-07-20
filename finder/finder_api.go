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
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/blueprint/pathtools"
)

// this file stores the portion of the Finder code that callers are expected to interact with directly

// a CacheParams specifies which files and directories the user wishes be scanned and potentially added to the cache
type CacheParams struct {
	RootDirs []string

	ExcludeDirs []string

	PruneFiles []string

	IncludeFiles []string
}

// a CacheConfig stores the inputs that determine what should be included in the cache
type CacheConfig struct {
	CacheParams

	Filesystem string
}

func (p *CacheConfig) Dump() ([]byte, error) {
	bytes, err := json.Marshal(p)
	return bytes, err
}

// Update getFinderVersion whenever making a backwards-incompatible change to the cache file format
func getVersionString() string {
	return "Android finder version 1"
}

// a FinderMetadata stores version information about the cache
type FinderMetadata struct {
	// The Version enables the Finder to determine whether it can even parse the file
	// If the version changes, the entire cache file must be regenerated
	Version string

	// The CacheParams enables the Finder to determine whether the parameters match
	// If the CacheParams change, the Finder can choose how much of the cache file to reuse
	// (although in practice, the Finder will probably choose to ignore the entire file anyway)
	Config CacheConfig
}

// the Finder is the main struct that callers will want to use
type Finder struct {
	// configuration
	numDbLoadingThreads int
	numPollingThreads   int
	numSearchingThreads int
	metadata            FinderMetadata
	dbPath              string
	logger              log.Logger
	filesystem          pathtools.FileSystem
	workingDir          string

	// temporary state
	statResponses      chan *pathAndStats
	childListResponses chan *DirEntries
	requests           sync.WaitGroup

	// non-temporary state
	modifiedFlag int32
	nodes        pathMap
}

// New creates a new Finder for use
func New(cacheParams CacheParams, filesystem pathtools.FileSystem, logger log.Logger, dbPath string) (*Finder, error) {

	numThreads := runtime.NumCPU() * 2
	numDbLoadingThreads := numThreads
	numPollingThreads := numThreads/30 + 1
	numSearchingThreads := numThreads - numPollingThreads
	if numSearchingThreads < 1 {
		numSearchingThreads = 1
	}

	metadata := FinderMetadata{
		Version: getVersionString(),
		Config: CacheConfig{
			CacheParams: cacheParams,
			Filesystem:  filesystem.Id(),
		},
	}

	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "/tmp"
	}

	finder := &Finder{
		numDbLoadingThreads: numDbLoadingThreads,
		numPollingThreads:   numPollingThreads,
		numSearchingThreads: numSearchingThreads,
		metadata:            metadata,
		logger:              logger,
		filesystem:          filesystem,
		workingDir:          workingDir,

		statResponses:      make(chan *pathAndStats, 100+numThreads),
		childListResponses: make(chan *DirEntries, 100+numThreads),
		requests:           sync.WaitGroup{},

		nodes:  *newPathMap(),
		dbPath: dbPath,
	}

	err = finder.loadDb()
	if err != nil {
		return nil, err
	}
	finder.Verbosef("done parsing db\n")
	finder.pollForStatResponses()
	return finder, nil
}

// FindNamed searches under <rootPath> for every file named <fileName>
func (f *Finder) FindNamed(rootPath string, fileName string) []string {
	scanStart := time.Now()

	isRel := !filepath.IsAbs(rootPath)

	workingDir := f.workingDir
	if isRel {
		rootPath = filepath.Join(workingDir, rootPath)
	}
	rootPath = filepath.Clean(rootPath)

	f.prepareToFind(rootPath)

	f.Verbosef("finder finding %v using cache\n", rootPath)

	matching := func(filePath string) bool {
		_, leaf := filepath.Split(filePath)
		return leaf == fileName
	}

	channel := make(chan []string, 1)
	f.findInCacheMultithreaded(rootPath, matching, f.numSearchingThreads, channel)
	results := <-channel
	close(channel)

	if isRel {
		for i := 0; i < len(results); i++ {
			results[i] = strings.Replace(results[i], workingDir+"/", "", 1)
		}
	}

	sort.Strings(results)
	f.Verbosef("found %v files under %v in %v using cache\n", len(results), rootPath, time.Now().Sub(scanStart))

	return results
}

// Shutdown saves the contents of the Finder to its database file
func (f *Finder) Shutdown() {
	f.Verbosef("shutting down\n")
	f.WaitUntilIdle()
	if f.wasModified() {
		err := f.dumpDb()
		if err != nil {
			f.Verbosef("%v\n", err)
		}
	} else {
		f.Verbosef("Skipping dumping unmodified db\n")
	}
}

// StartFind asks that the finder begin to load the contents of <path> into its cache
func (f *Finder) StartFind(path string) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(f.workingDir, path)
	}
	f.statDirAsync(path)
}

func (f *Finder) WaitUntilIdle() {
	startDate := time.Now()
	f.Verbosef("waiting for pending requests to complete\n")
	f.requests.Wait()
	f.Verbosef("is idle after %v\n", time.Now().Sub(startDate))
}

func (f *Finder) SetWorkingDir(path string) {
	f.workingDir = filepath.Clean(path)
}
