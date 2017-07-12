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
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// This file stores the portion of the Finder code that is relevant to callers

// a CacheParams specifies which files and directories should be scanned and potentially added to the cache
type CacheParams struct {
	RootDirs []string

	PruneDirs  []string
	PruneFiles []string

	IncludeFiles []string
}

// a CacheConfig stores the inputs used by the finder to determine what to include in the cache
type CacheConfig struct {
	CacheParams

	Host string
	User string
}

func (p *CacheConfig) Dump() ([]byte, error) {
	bytes, err := json.Marshal(p)
	return bytes, err
}

// Update getFinderVersion whenever making a backwards-incompatible change to the cache file format
func getVersionString() string {
	return "finder.go version 1"
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
	statResponses      chan *pathAndStats
	childListResponses chan *DirEntries

	nodes  pathMap
	dbPath string

	requests     sync.WaitGroup
	modifiedFlag int32

	numDbLoadingThreads int
	numPollingThreads   int
	numSearchingThreads int

	metadata FinderMetadata
}

// New creates a new Finder for use
func New(cacheParams CacheParams, dbPath string) (*Finder, error) {

	numThreads := runtime.NumCPU() * 2
	numDbLoadingThreads := numThreads
	numPollingThreads := numThreads/30 + 1
	numSearchingThreads := numThreads - numPollingThreads
	if numSearchingThreads < 1 {
		numSearchingThreads = 1
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	username := user.Username

	metadata := FinderMetadata{
		Version: getVersionString(),
		Config: CacheConfig{
			User:        username,
			Host:        hostname,
			CacheParams: cacheParams,
		},
	}

	finder := &Finder{

		statResponses:      make(chan *pathAndStats, 100+numThreads),
		childListResponses: make(chan *DirEntries, 100+numThreads),

		nodes:  *newPathMap(),
		dbPath: dbPath,

		requests: sync.WaitGroup{},

		numDbLoadingThreads: numDbLoadingThreads,
		numPollingThreads:   numPollingThreads,
		numSearchingThreads: numSearchingThreads,

		metadata: metadata,
	}

	err = finder.loadDb()
	if err != nil {
		return nil, err
	}
	fmt.Printf("done parsing db\n")
	finder.pollForStatResponses()
	return finder, nil
}

// FindNamed searches under <rootPath> for every file named <fileName>
func (f *Finder) FindNamed(rootPath string, fileName string) []string {
	scanStart := time.Now()
	fmt.Printf("finder finding %v using cache\n", rootPath)

	f.prepareToFind(rootPath)

	matching := func(filePath string) bool {
		_, leaf := filepath.Split(filePath)
		return leaf == fileName
	}
	channel := f.findInCacheMultithreaded(rootPath, matching, f.numSearchingThreads)
	results := <-channel
	fmt.Printf("found %v files under %v in %v using cache\n", len(results), rootPath, time.Now().Sub(scanStart))

	return results
}

// Shutdown saves the contents of the Finder to its database file
func (f *Finder) Shutdown() {
	fmt.Printf("shutting down\n")
	f.WaitUntilIdle()
	if f.wasModified() {
		f.dumpDb()
	} else {
		fmt.Printf("Skipping dumping unmodified db\n")
	}
}

// StartFind asks that the finder begin to load the contents of <path> into its cache
func (f *Finder) StartFind(path string) {
	f.statDirAsync(path)
}

func (f *Finder) WaitUntilIdle() {
	startDate := time.Now()
	fmt.Printf("waiting for pending requests to complete\n")
	f.requests.Wait()
	fmt.Printf("Is idle after %v\n", time.Now().Sub(startDate))
}
