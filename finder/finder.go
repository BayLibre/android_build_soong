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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bytes"

	"github.com/google/blueprint/pathtools"
)

// This file provides a Finder struct that can quickly search for files satisfying
// certain criteria.
// This Finder gets its speed partially from parallelism and partially from caching.
// If a Stat call returns the same result as last time, then it means Finder
// can skip the ReadDir call for that dir

// The primary data structure used by the finder is the field Finder.nodes ,
// which is a tree of nodes of type *pathMap .
// Each node represents a directory on disk, along with its stats, subdirectories,
// and contained files.

// The common use case for the Finder is that the caller creates a Finder and gives
// it the same query that was given to it in the previous execution.
// In this situation, the major events that take place are:
// 1. The Finder begins to load its db
// 2. The Finder begins to stat the directories mentioned in its db (using multiple threads)
//    Calling Stat on each of these directories is generally a large fraction of the total time
// 3. The Finder begins to construct a tree separate tree of nodes in each of its threads
// 4. The Finder may call ReadDir a few times if there are a few directories that are out-of-date
// 5. The Finder merges the individual node trees into the main node tree
// 6. The Finder waits for all loading to complete
// 7. The Finder searches the cache for files matching the user's query (using multiple threads)

// These are the invariants regarding concurrency:
// 1. Only one function outside of the Finder may access the Finder at any one time
//    This is implemented via a Mutex.
//    This restriction can be relaxed in the future if the need arises.
// 2. Only one thread may ever modify the <children> collection of a *pathMap at once
//    The thread that modifies the <children> collection is the thread that discovers the children
//    (by reading them from the cache or by having received a response to ReadDir)
// 3. No query may be begun until all loading (both reading the db
//    and scanning the filesystem) is complete
//    Tests indicate that it only takes about 10% as long to search the in-memory cache as to
//    generate it, making this not a huge loss in performance
// 4. Only one thread at a time may merge its node tree into the main node tree

// see cmd/finder.go or finder_test.go for usage examples

// a CacheParams specifies which files and directories the user wishes be scanned and
// potentially added to the cache
type CacheParams struct {
	// RootDirs are the root directories used to initiate the search
	RootDirs []string

	// ExcludeDirs are directory names that if encountered are removed from the search
	ExcludeDirs []string

	// PruneFiles are file names that if encountered prune their entire directory
	// (including siblings)
	PruneFiles []string

	// IncludeFiles are file names to include as matches
	IncludeFiles []string
}

// a cacheConfig stores the inputs that determine what should be included in the cache
type cacheConfig struct {
	CacheParams

	Filesystem string
}

func (p *cacheConfig) Dump() ([]byte, error) {
	bytes, err := json.Marshal(p)
	return bytes, err
}

// Update getFinderVersion whenever making a backwards-incompatible change to the cache file format
func getVersionString() string {
	return "Android finder version 1"
}

// a finderMetadata stores version information about the cache
type finderMetadata struct {
	// The Version enables the Finder to determine whether it can even parse the file
	// If the version changes, the entire cache file must be regenerated
	Version string

	// The CacheParams enables the Finder to determine whether the parameters match
	// If the CacheParams change, the Finder can choose how much of the cache file to reuse
	// (although in practice, the Finder will probably choose to ignore the entire file anyway)
	Config cacheConfig
}

// the Finder is the main struct that callers will want to use
type Finder struct {
	// configuration
	DbPath              string
	numDbLoadingThreads int
	numSearchingThreads int
	metadata            finderMetadata
	logger              log.Logger
	filesystem          pathtools.FileSystem
	workingDir          string

	// temporary state
	statResponses chan *pathAndStats
	requests      sync.WaitGroup
	mutex         sync.Mutex

	// non-temporary state
	modifiedFlag int32
	nodes        pathMap
}

// New creates a new Finder for use
func New(cacheParams CacheParams, filesystem pathtools.FileSystem,
	logger log.Logger, dbPath string) *Finder {

	numThreads := runtime.NumCPU() * 2
	numDbLoadingThreads := numThreads
	numSearchingThreads := numThreads

	metadata := finderMetadata{
		Version: getVersionString(),
		Config: cacheConfig{
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
		numSearchingThreads: numSearchingThreads,
		metadata:            metadata,
		logger:              logger,
		filesystem:          filesystem,
		workingDir:          workingDir,

		statResponses: make(chan *pathAndStats, 100+numThreads),
		requests:      sync.WaitGroup{},

		nodes:  *newPathMap("/"),
		DbPath: dbPath,
	}

	if !finder.startFromExternalCache() {
		finder.startWithoutExternalCache()
	}

	finder.Verbosef("done parsing db\n")
	return finder
}

// FindNamed searches under <rootPath> for every file named <fileName>
func (f *Finder) FindNamed(rootPath string, fileName string) []string {
	predicate := func(filePath string) bool {
		_, leaf := filepath.Split(filePath)
		return leaf == fileName
	}
	return f.FindMatching(rootPath, predicate)
}

func (f *Finder) FindMatching(rootPath string, predicate func(filePath string) bool) []string {
	// setup some parameters
	scanStart := time.Now()
	var isRel bool
	workingDir := f.workingDir

	isRel = !filepath.IsAbs(rootPath)
	if isRel {
		rootPath = filepath.Join(workingDir, rootPath)
	}

	rootPath = filepath.Clean(rootPath)

	// ensure nothing else is using the Finder
	f.Verbosef("Find waiting for finder to be idle\n")
	f.lock()
	defer f.unlock()
	f.waitUntilIdle()

	if !f.hasFreshData(rootPath) {
		f.Verbosef("Path %v is not included in cache params\n", rootPath)
		return []string{}
	}

	// search for matching files
	f.Verbosef("finder finding %v using cache\n", rootPath)
	channel := make(chan []string, 1)
	f.findInCacheMultithreaded(rootPath, predicate, f.numSearchingThreads, channel)
	results := <-channel
	close(channel)

	// format and return results
	if isRel {
		for i := 0; i < len(results); i++ {
			results[i] = strings.Replace(results[i], workingDir+"/", "", 1)
		}
	}
	sort.Strings(results)
	f.Verbosef("found %v files under %v in %v using cache\n",
		len(results), rootPath, time.Now().Sub(scanStart))
	return results
}

// Shutdown saves the contents of the Finder to its database file
func (f *Finder) Shutdown() {
	f.Verbosef("shutting down\n")
	f.waitUntilIdle()
	if f.wasModified() {
		err := f.dumpDb()
		if err != nil {
			f.Verbosef("%v\n", err)
		}
	} else {
		f.Verbosef("Skipping dumping unmodified db\n")
	}
	close(f.statResponses)
}

func (f *Finder) SetWorkingDir(path string) {
	f.lock()
	f.waitUntilIdle()
	f.workingDir = filepath.Clean(path)
	f.unlock()
}

// End of public api

func (f *Finder) startFind(path string) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(f.workingDir, path)
	}
	node := f.nodes.GetNode(path, true)
	f.statDirAsync(node)
}

func (f *Finder) lock() {
	f.mutex.Lock()
}

func (f *Finder) unlock() {
	f.mutex.Unlock()
}

func (f *Finder) waitUntilIdle() {
	startDate := time.Now()
	f.Verbosef("waiting for pending requests to complete\n")
	f.requests.Wait()
	f.Verbosef("is idle after %v\n", time.Now().Sub(startDate))
}

var lineSeparator = byte('\n')

// a statResponse is the relevant portion of the response from the filesystem to a Stat call
type statResponse struct {
	ModTime int64
	Inode   uint64
	Device  uint64
}

// a pathAndStats stores a path and its stats
type pathAndStats struct {
	statResponse

	Path string
}

// a dirFullInfo stores all of the relevant information we know about a directory
type dirFullInfo struct {
	pathAndStats

	FileNames []string
}

// a PersistedDirInfo is the information about a dir that we save to our cache on disk
type PersistedDirInfo struct {
	// These field names are short because they are repeated many times in the output json file
	P string   // path
	T int64    // mod time
	I uint64   // inode number
	F []string // relevant files contained
}

// a PersistedDirs is the information that we persist for a group of dirs
type PersistedDirs struct {
	Device uint64
	Root   string
	Dirs   []PersistedDirInfo
}

// a CacheEntry is the smallest unit that can be read and parsed from the cache (on disk) at a time
type CacheEntry []PersistedDirs

// a DirEntries lists the files and directories contained directly within a specific directory
type DirEntries struct {
	Path string

	// elements of SubDirs are just the dir names, that is, they don't include any path separator ("/")
	SubDirs []string
	// elements of FileNames are just the file names, that is, they don't include any path separator ("/")
	Files []string
}

// a mapNode stores the relevant stats about a directory to be stored in a pathMap
type mapNode struct {
	statResponse
	FileNames []string
}

// a pathMap implements the tree structure of nodes
type pathMap struct {
	mapNode

	path string

	children map[string]*pathMap

	approximateNumDescendents int
}

// A pathMap always is associated with a single path
// However, if we allocate a path for each pathMap, then that approximately triples the total runtime
// (recall that this finder is supposed to be very fast).
// So we provide a labeledPathMap for cases where both the path and pathMap are needed at once
//type labeledPathMap pathMap

//type labeledPathMap struct {
//	path string
//	node *pathMap
//}

func newPathMap(path string) *pathMap {
	result := &pathMap{path: path, children: make(map[string]*pathMap, 4), approximateNumDescendents: 1}
	return result
}

// GetNode returns the node at <path>
func (m *pathMap) GetNode(path string, createIfNotFound bool) *pathMap {
	components := m.pathComponents(path)
	result, err := m.getNodeForComponents(components, createIfNotFound)
	if err != nil {
		panic(fmt.Sprintf("failed to get node for path %v with error %v\n", path, err))
	}
	return result
}

func (m *pathMap) getNodeForComponents(components []string, createIfNotFound bool) (*pathMap, error) {
	if len(components) == 0 {
		return m, nil
	}

	childComponent := components[0]
	if len(childComponent) < 1 {
		return nil, fmt.Errorf("illegal component '%v' at index 0 of components %v", childComponent, components)
	}

	subMap, found := m.children[childComponent]

	if !found {
		if createIfNotFound {
			m.newChild(childComponent)
			subMap = m.children[childComponent]
		} else {
			return nil, nil
		}
	}
	return subMap.getNodeForComponents(components[1:], createIfNotFound)
}

func (m *pathMap) newChild(name string) (child *pathMap) {
	var path string
	if m.path == "/" {
		path = "/" + name
	} else {
		path = m.path + "/" + name
	}
	newChild := newPathMap(path)
	m.children[name] = newChild

	return m.children[name]
}

func (m *pathMap) UpdateNumDescendents() int {
	count := 1
	for _, child := range m.children {
		count += child.approximateNumDescendents
	}
	m.approximateNumDescendents = count
	return count
}

func (m *pathMap) UpdateNumDescendentsRecursive() {
	for _, child := range m.children {
		child.UpdateNumDescendentsRecursive()
	}
	m.UpdateNumDescendents()
}

func (m *pathMap) MergeIn(other *pathMap) {
	for key, theirs := range other.children {
		ours, found := m.children[key]
		if found {
			ours.MergeIn(theirs)
		} else {
			m.children[key] = theirs
		}
	}
	if other.ModTime != 0 {
		m.mapNode = other.mapNode
	}
	m.UpdateNumDescendents()
}

func (m *pathMap) pathComponents(path string) (components []string) {
	if path == "" {
		return []string{}
	}
	//path = filepath.Clean(path)
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}

func (m *pathMap) DumpAll() []dirFullInfo {
	results := []dirFullInfo{}
	m.dumpInto("", &results)
	return results
}

func (m *pathMap) dumpInto(path string, results *[]dirFullInfo) {
	*results = append(*results,
		dirFullInfo{
			pathAndStats{statResponse: m.statResponse, Path: path},
			m.FileNames},
	)
	for key, child := range m.children {
		childPath := path + "/" + key
		child.dumpInto(childPath, results)
	}
}

func (f *Finder) serializeCacheEntry(dirInfos []dirFullInfo) ([]byte, error) {
	// group each dirFullInfo by its Device, to reduce repetition
	dirsByDevice := map[uint64][]PersistedDirInfo{}
	for _, entry := range dirInfos {
		_, found := dirsByDevice[entry.Device]
		if !found {
			dirsByDevice[entry.Device] = []PersistedDirInfo{}
		}
		dirsByDevice[entry.Device] = append(dirsByDevice[entry.Device],
			PersistedDirInfo{P: entry.Path, T: entry.ModTime, I: entry.Inode, F: entry.FileNames})
	}

	// turn the map into a list of structs with labeled fields, for readability
	cacheEntry := CacheEntry{}

	for device, infos := range dirsByDevice {
		// find common prefix
		prefix := ""
		if len(infos) > 0 {
			prefix = infos[0].P
		}
		for _, info := range infos {
			for !strings.HasPrefix(info.P+"/", prefix+"/") {
				prefix = filepath.Dir(prefix)
			}
		}
		// remove common prefix
		for i := range infos {
			suffix := strings.Replace(infos[i].P, prefix, "", 1)
			if len(suffix) > 0 && suffix[0] == '/' {
				suffix = suffix[1:]
			}
			infos[i].P = suffix
		}
		cacheEntry = append(cacheEntry, PersistedDirs{Device: device, Root: prefix, Dirs: infos})
	}

	// convert to json.
	// it would save some space to use a different format than json for the db file,
	// but the space and time savings are small, and json is easy for humans to read
	bytes, err := json.Marshal(cacheEntry)
	return bytes, err
}

func (f *Finder) parseCacheEntry(bytes []byte) ([]dirFullInfo, error) {
	var cacheEntry CacheEntry
	err := json.Unmarshal(bytes, &cacheEntry)
	if err != nil {
		return nil, err
	}

	// convert from a CacheEntry to a []dirFullInfo (by copying a few fields)
	var nodes []dirFullInfo
	for _, element := range cacheEntry {
		for _, dir := range element.Dirs {
			path := element.Root
			// we could call filepath.Join but it runs slightly slower by checking for more things (like "..")
			if path != "/" && dir.P != "" {
				path = path + "/" + dir.P
			} else {
				path = path + dir.P
			}

			nodes = append(nodes,
				dirFullInfo{pathAndStats: pathAndStats{statResponse: statResponse{
					ModTime: dir.T, Inode: dir.I, Device: element.Device},
					Path: path},
					FileNames: dir.F})
		}
	}
	return nodes, nil
}

func (f *Finder) Verbosef(format string, args ...interface{}) {
	f.logger.Output(2, fmt.Sprintf(format, args...))
}

// validateCacheHeader reads the cache header from cacheReader and tells whether the cache is compatible with this Finder
func (f *Finder) validateCacheHeader(cacheReader *bufio.Reader) bool {
	cacheVersionBytes, err := cacheReader.ReadBytes(lineSeparator)
	if err != nil {
		f.Verbosef("failed to read database header; database is invalid\n")
		return false
	}
	if len(cacheVersionBytes) > 0 && cacheVersionBytes[len(cacheVersionBytes)-1] == lineSeparator {
		cacheVersionBytes = cacheVersionBytes[:len(cacheVersionBytes)-1]
	}
	cacheVersionString := string(cacheVersionBytes)
	currentVersion := f.metadata.Version
	if cacheVersionString != currentVersion {
		f.Verbosef("version changed from %q to %q, database is not applicable\n", cacheVersionString, currentVersion)
		return false
	}

	cacheParamBytes, err := cacheReader.ReadBytes(lineSeparator)
	if err != nil {
		f.Verbosef("failed to read database search params; database is invalid\n")
		return false
	}
	if len(cacheParamBytes) > 0 && cacheParamBytes[len(cacheParamBytes)-1] == lineSeparator {
		cacheParamBytes = cacheParamBytes[:len(cacheParamBytes)-1]
	}

	currentParamBytes, err := f.metadata.Config.Dump()
	if err != nil {
		panic("finder failed to serialize its parameters")
	}
	cacheParamString := string(cacheParamBytes)
	currentParamString := string(currentParamBytes)
	if cacheParamString != currentParamString {
		f.Verbosef("params changed from %q to %q, database is not applicable\n", cacheParamString, currentParamString)
		return false
	}
	return true
}

// loadBytes compares the data in <data> to the state of the filesystem
// loadBytes returns a map representing <data> and also a slice of dirs that need to be re-walked
func (f *Finder) loadBytes(description string, data []byte) (m *pathMap, dirsToWalk []string, err error) {

	helperStartDate := time.Now()

	cachedNodes, err := f.parseCacheEntry(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse %v: %v\n", description, err.Error())
	}

	unmarshalDate := time.Now()
	f.Verbosef("unmarshaled %v objects for %v in %v\n", len(cachedNodes), description, unmarshalDate.Sub(helperStartDate))

	tempMap := newPathMap("/")
	stats := make([]statResponse, len(cachedNodes))
	for i, node := range cachedNodes {
		// check the file system for an updated timestamp
		stats[i] = f.statDirSync(node.Path)

	}

	dirsToWalk = []string{}
	for i, cachedNode := range cachedNodes {
		updated := stats[i]
		// save the cached value
		container := tempMap.GetNode(cachedNode.Path, true)
		container.mapNode = mapNode{statResponse: cachedNode.statResponse, FileNames: cachedNode.FileNames}

		// if the metadata changed and the directory still exists, then make a note to walk it later
		if !f.isInfoUpToDate(cachedNode.statResponse, updated) && updated.ModTime != 0 {
			// make a note that the directory needs to be walked
			dirsToWalk = append(dirsToWalk, cachedNode.Path)
		}
	}
	// count the number of nodes to improve our understanding of the shape of the tree,
	// thereby improving performance of subsequent searches
	tempMap.UpdateNumDescendentsRecursive()
	// merge our temp map back into the main map

	f.Verbosef("statted inodes of %v in %v\n", description, time.Now().Sub(unmarshalDate))
	return tempMap, dirsToWalk, nil
}

// startFromExternalCache loads the cache database from disk
// startFromExternalCache waits to return until the load of the cache sortedDirEntries is complete, but
// startFromExternalCache does not wait for all pending listDir() or statDir() or requests to complete
func (f *Finder) startFromExternalCache() (loaded bool) {
	startDate := time.Now()
	f.waitUntilIdle()
	dbPath := f.DbPath

	// open cache file and validate its header
	reader, err := f.filesystem.Open(dbPath)
	if err != nil {
		f.Verbosef("no data to load from database\n")
		return false
	}
	bufferedReader := bufio.NewReader(reader)
	if !f.validateCacheHeader(bufferedReader) {
		return false
	}
	f.Verbosef("database header matches, will attempt to use database %v\n", f.DbPath)

	statusChannel := make(chan bool, f.numDbLoadingThreads)
	numStartedThreads := 0
	numCompletedThreads := 0

	// read the file and spawn threads to process it
	reading := true
	waiting := true
	mapLock := &sync.Mutex{}
	nodesToWalk := [][]*pathMap{}
	successful := true
	for reading || waiting {
		if reading {
			// spawn threads until we have as many as allowed
			prevNumStartedThreads := numStartedThreads
			spawnStart := time.Now()
			for numStartedThreads < numCompletedThreads+f.numDbLoadingThreads {
				// It takes some time to unmarshal the input from json, so we want to unmarshal it in parallel.
				// In order to find valid places to break the input, we scan for the newlines that we
				// inserted (for this purpose) when we dumped the database.
				nextBlock, err := bufferedReader.ReadBytes(lineSeparator)
				if err != nil && err != io.EOF {
					// failed to read block, give up trying to read db, and mark it as failed
					statusChannel <- false
					reading = false
					waiting = false
					break
				}

				// spawn a goroutine to process this block
				go func(i int, block []byte) {
					description := fmt.Sprintf("block %v", i)
					f.Verbosef("starting to process %v after %v\n", description, time.Now().Sub(startDate))
					tempMap, updatedDirs, err := f.loadBytes(description, block)
					threadSuccessful := true
					if err != nil {
						f.Verbosef("%v failed to parse with error %v\n", description, err)
						threadSuccessful = false
					} else {
						mapLock.Lock()
						f.nodes.MergeIn(tempMap)
						updatedNodes := make([]*pathMap, len(updatedDirs))
						for j, dir := range updatedDirs {
							node := f.nodes.GetNode(dir, false)
							updatedNodes[j] = node
						}
						nodesToWalk = append(nodesToWalk, updatedNodes)
						mapLock.Unlock()
					}
					statusChannel <- threadSuccessful
				}(numStartedThreads, nextBlock)
				numStartedThreads++

				// stop at end of file
				if err == io.EOF {
					reading = false
					break
				}
			}
			f.Verbosef("spawned %v db-processing threads in %v\n", numStartedThreads-prevNumStartedThreads, time.Now().Sub(spawnStart))

		}
		if numCompletedThreads < numStartedThreads {
			f.Verbosef("waiting for %v db-processing threads to complete\n", numStartedThreads-numCompletedThreads)
			// count completed threads and potentially spawn replacement threads
			threadSucceeded := <-statusChannel
			numCompletedThreads++

			if !threadSucceeded {
				reading = false
				successful = false
			}
		} else {
			waiting = false
		}
	}

	if !successful {
		// ignore the db if it failed to parse
		f.waitUntilIdle()
		f.nodes = *newPathMap("/")
	}

	// after having loaded the entire db and therefore created entries for the directories we know of,
	// now it's safe to start calling ReadDir on any updated directories
	for i := range nodesToWalk {
		f.listDirsAsync(nodesToWalk[i])
	}
	f.Verbosef("loaded db and statted its contents in %v\n", time.Now().Sub(startDate))
	return successful
}

// startWithoutExternalCache starts scanning the filesystem according to the cache config
// startWithoutExternalCache should be called if startFromExternalCache is not applicable
func (f *Finder) startWithoutExternalCache() {
	configDirs := f.metadata.Config.RootDirs

	// clean paths
	candidates := make([]string, len(configDirs))
	for i, dir := range configDirs {
		candidates[i] = filepath.Clean(dir)
	}
	// remove duplicates
	dirsToScan := make([]string, 0, len(configDirs))
	for _, candidate := range candidates {
		include := true
		for _, included := range dirsToScan {
			if included == "/" || strings.HasPrefix(candidate+"/", included+"/") {
				include = false
				break
			}
		}
		if include {
			dirsToScan = append(dirsToScan, candidate)
		}
	}

	// start searching finally
	for _, path := range dirsToScan {
		f.Verbosef("starting find of %v\n", path)
		f.startFind(path)
	}
}

// isInfoUpToDate tells whether <new> can confirm that results computed at <old> are still valid
func (f *Finder) isInfoUpToDate(old statResponse, new statResponse) (equal bool) {
	if old.Inode != new.Inode {
		return false
	}
	if old.ModTime != new.ModTime {
		return false
	}
	return true
}

func (f *Finder) wasModified() bool {
	return f.modifiedFlag > 0
}

func (f *Finder) setModified(modified bool) {
	var newVal int32
	if modified {
		newVal = 1
	} else {
		newVal = 0
	}
	atomic.StoreInt32(&f.modifiedFlag, newVal)
}

// sortedDirEntries exports directory entries to facilitate dumping them to the external cache
func (f *Finder) sortedDirEntries() []dirFullInfo {
	startDate := time.Now()
	nodes := make([]dirFullInfo, 0)
	for _, node := range f.nodes.DumpAll() {
		if node.ModTime != 0 {
			nodes = append(nodes, node)
		}
	}
	discoveryDate := time.Now()
	f.Verbosef("generated %v cache entries in %v\n", len(nodes), discoveryDate.Sub(startDate))
	less := func(i int, j int) bool {
		return nodes[i].Path < nodes[j].Path
	}
	sort.Slice(nodes, less)
	sortDate := time.Now()
	f.Verbosef("sorted %v cache entries in %v\n", len(nodes), sortDate.Sub(discoveryDate))

	return nodes
}

// serializeDb converts the cache database into a form to save to disk
func (f *Finder) serializeDb() ([]byte, error) {
	// sort dir entries
	var entryList = f.sortedDirEntries()

	// Generate an output file that can be conveniently loaded using the same number of threads as were
	// used in this execution.

	// generate header
	header := []byte{}
	header = append(header, []byte(f.metadata.Version)...)
	header = append(header, lineSeparator)
	configDump, err := f.metadata.Config.Dump()
	if err != nil {
		return nil, err
	}
	header = append(header, configDump...)

	// serialize individual blocks in parallel
	numBlocks := f.numDbLoadingThreads
	if numBlocks > len(entryList) {
		numBlocks = len(entryList)
	}
	blocks := make([][]byte, 1+numBlocks)
	blocks[0] = header
	blockMin := 0
	wg := sync.WaitGroup{}
	var errLock sync.Mutex

	for i := 1; i <= numBlocks; i++ {
		// identify next block
		blockMax := len(entryList) * i / numBlocks
		block := entryList[blockMin:blockMax]

		// process block
		wg.Add(1)
		go func(index int, block []dirFullInfo) {
			byteBlock, subErr := f.serializeCacheEntry(block)
			f.Verbosef("size of block %v = %v\n", index, len(byteBlock))
			if subErr != nil {
				errLock.Lock()
				err = subErr
				errLock.Unlock()
			} else {
				blocks[index] = byteBlock
			}
			wg.Done()
		}(i, block)

		blockMin = blockMax
	}

	wg.Wait()

	if err != nil {
		return nil, err
	}

	content := bytes.Join(blocks, []byte{lineSeparator})

	return content, nil
}

// dumpDb saves the cache database to disk
func (f *Finder) dumpDb() error {
	startDate := time.Now()
	f.Verbosef("dumping db\n")

	tempPath := f.DbPath + ".tmp"

	bytes, err := f.serializeDb()
	if err != nil {
		return err
	}
	serializeDate := time.Now()
	f.Verbosef("serialized db in %v\n", serializeDate.Sub(startDate))
	// dump file and atomically move
	err = f.filesystem.WriteFile(tempPath, bytes, 0777)
	if err != nil {
		return err
	}
	err = f.filesystem.Rename(tempPath, f.DbPath)
	if err != nil {
		return err
	}

	f.Verbosef("wrote db in %v\n", time.Now().Sub(serializeDate))
	return nil
}

func (f *Finder) statDirAsync(dir *pathMap) {
	node := dir
	path := dir.path
	f.requests.Add(1)

	go func() {
		updatedStats := f.statDirSync(path)

		if !f.isInfoUpToDate(node.statResponse, updatedStats) {
			node.mapNode = mapNode{statResponse: updatedStats, FileNames: []string{}}
			if node.statResponse.ModTime != 0 {
				// modification time was updated, so re-scan for child directories
				f.listDirAsync(dir)
			} else {
				// directory was deleted; don't have to rescan it but do have to note modification
				f.setModified(true)
			}
		}
		f.requests.Done()

	}()
}

func (f *Finder) statDirSync(path string) statResponse {
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	fileInfo, err := f.filesystem.Lstat(path)
	// in case of any error, pass an empty response into the channel

	var stats statResponse
	if err == nil {
		stats = statResponse{ModTime: fileInfo.ModTime().UnixNano()}
		inode, err := f.filesystem.InodeNumber(fileInfo)
		if err == nil {
			stats.Inode = inode
		}
		device, err := f.filesystem.DeviceNumber(fileInfo)
		if err == nil {
			stats.Device = device
		}
	}

	return stats
}

// pruneCacheCandidates removes the items that we don't want to include in our persistent cache
func (f *Finder) pruneCacheCandidates(items *DirEntries) {
	writeIndex := 0
	for _, fileName := range items.Files {
		// include only these files
		for _, includedName := range f.metadata.Config.IncludeFiles {
			if fileName == includedName {
				items.Files[writeIndex] = fileName
				writeIndex++
				break
			}
		}
		// the presence of these files will completely terminate the search
		// (these files should be placed into an output directory)
		for _, abortedName := range f.metadata.Config.PruneFiles {
			if fileName == abortedName {
				items.Files = []string{}
				items.SubDirs = []string{}
				return
			}
		}
	}
	// resize
	items.Files = items.Files[:writeIndex]

	writeIndex = 0
	for _, dirName := range items.SubDirs {
		items.SubDirs[writeIndex] = dirName
		// ignore other dirs that are known to not be inputs to the build process
		include := true
		for _, excludedName := range f.metadata.Config.ExcludeDirs {
			if dirName == excludedName {
				// don't include
				include = false
				break
			}
		}
		if include {
			writeIndex++
		}
	}
	// resize
	items.SubDirs = items.SubDirs[:writeIndex]
}

func (f *Finder) listDirsAsync(nodes []*pathMap) {
	f.requests.Add(1)
	go func() {
		for i := range nodes {
			f.listDirSync(nodes[i])
		}
		f.requests.Done()
	}()
}

func (f *Finder) listDirAsync(node *pathMap) {
	f.requests.Add(1)
	go func() {
		f.listDirSync(node)
		f.requests.Done()
	}()
}

func (f *Finder) listDirSync(dir *pathMap) {
	path := dir.path
	children, _ := f.filesystem.ReadDir(path)

	var subdirs []string
	var subfiles []string

	for _, child := range children {
		linkBits := child.Mode() & os.ModeSymlink
		isLink := linkBits != 0
		if child.IsDir() {
			if !isLink {
				// Skip symlink dirs.
				// We don't have to support symlink dirs because that would cause duplicates.
				subdirs = append(subdirs, child.Name())
			}
		} else {
			// We do have to support symlink files because the link name might be different
			// than the target name
			// (for example, Android.bp -> build/soong/root.bp)
			subfiles = append(subfiles, child.Name())
		}

	}
	entry := &DirEntries{Path: path, SubDirs: subdirs, Files: subfiles}
	f.pruneCacheCandidates(entry)

	parentNode := dir

	// create a pathMap node for each relevant subdirectory
	relevantChildren := map[string]*pathMap{}
	for _, subdirName := range entry.SubDirs {
		childNode, found := parentNode.children[subdirName]
		// if we already knew of this directory, then we already have a request pending to Stat it
		// if we didn't already know of this directory, then we must Stat it now
		if !found {
			childNode = parentNode.newChild(subdirName)
			f.statDirAsync(childNode)
		}
		relevantChildren[subdirName] = childNode
	}
	parentNode.FileNames = entry.Files
	parentNode.children = relevantChildren

	f.setModified(true)
}

// hasFreshData tells whether the cache has an entry for the given path
func (f *Finder) hasFreshData(path string) bool {
	node := f.nodes.GetNode(path, false)
	if node == nil {
		return false
	}
	if node.ModTime == 0 {
		return false
	}
	return true
}

// findInCacheMultithreaded spawns potentially multiple goroutines with which to search the cache.
// findInCacheMultithreaded returns a channel that will yield a single item: the list of results.
func (f *Finder) findInCacheMultithreaded(path string, predicate func(string) bool, numThreads int, channelToUpdate chan []string) {
	if numThreads < 2 {
		f.findInCacheAsync(path, predicate, channelToUpdate)
		return
	}

	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	go func() {

		node := f.nodes.GetNode(path, false)
		totalWork := 0
		for _, child := range node.children {
			totalWork += child.approximateNumDescendents
		}
		childrenResults := make(chan []string, len(node.children))

		// check for results in the current directory
		results := f.listMatching(path, predicate, node)

		// spawn threads to process child directories
		for key, child := range node.children {
			numChildThreads := numThreads * child.approximateNumDescendents / totalWork
			f.findInCacheMultithreaded(path+"/"+key, predicate, numChildThreads, childrenResults)
		}
		// collect results
		for i := 0; i < len(node.children); i++ {
			childResults := <-childrenResults
			results = append(results, childResults...)
		}
		close(childrenResults)

		channelToUpdate <- results
	}()
}

// findInCacheAsync asynchronously searches the cache for all matching file paths
func (f *Finder) findInCacheAsync(path string, predicate func(string) bool, channelToUpdate chan []string) {
	go func() {
		results := f.findInCacheSync(path, predicate)
		channelToUpdate <- results
	}()
}

// findInCacheSync synchronously searches the cache for all matching file paths
func (f *Finder) findInCacheSync(path string, predicate func(string) bool) []string {
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	searchDirs := []string{path}
	startingMap := f.nodes.GetNode(path, false)
	if startingMap == nil {
		return []string{}
	}

	nodes := []*pathMap{startingMap}
	matches := []string{}

	// we avoid the use of 'range' because the length of <paths> will change while iterating
	i := 0
	for i < len(searchDirs) {
		currentPath := searchDirs[i]
		node := nodes[i]
		i++

		children := node.children
		for key, child := range children {
			childPath := currentPath + "/" + key
			searchDirs = append(searchDirs, childPath)
			nodes = append(nodes, child)
		}
		matches = append(matches, f.listMatching(currentPath, predicate, node)...)
	}
	return matches
}

// listMatching returns the absolute path of files in the given directory that match the given predicate
func (f *Finder) listMatching(path string, predicate func(string) bool, node *pathMap) []string {
	matches := []string{}
	for _, fileName := range node.FileNames {
		filePath := path + "/" + fileName
		if predicate(filePath) {
			matches = append(matches, filePath)
		}
	}
	return matches
}
