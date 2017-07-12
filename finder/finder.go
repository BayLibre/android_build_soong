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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type dataFreshness int

const (
	noData = iota
	staleData
	currentData
)

type statResponse struct {
	dirs []StattedDir
}
type StattedDir struct {
	P string // short name because it gets copied into the output file
	DirStats
}

func Path(d *StattedDir) string {
	return d.P
}

type DirStats struct {
	T int64 // short name because it gets copied into the output file
}

type DirFullInfo struct {
	StattedDir
	F []string // Files
}

func (s *DirStats) ModTime() int64 {
	return s.T
}

type childrenSnapshot struct {
	content *ChildList
}

type ChildList struct {
	Path    string
	SubDirs []string
	Files   []string
}

type mapNode struct {
	DirStats
	Files []string
}

// note that Finder is not threadsafe from the perspective of its callers, even though it internally uses channels
// TODO clean up some parallelism corner cases in this file
type Finder struct {
	filterDir func(items *ChildList)

	statResponses      chan *statResponse
	childListResponses chan *ChildList

	nodes pathMap
	//nodes    map[string]StattedDir
	//children map[string]childrenSnapshot
	dbPath string

	requests sync.WaitGroup
	//numIssuedRequests   int
	//numServicedRequests int
	modifiedFlag int32
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

func (f *Finder) loadDb() error {
	startDate := time.Now()
	f.WaitUntilIdle()
	dbPath := f.dbPath
	//fmt.Printf("finder loading db %s\n", dbPath)

	// read file
	//var nodeSnapshots []StattedDir
	reader, err := os.Open(dbPath)
	if err != nil {
		// if the file is not found, then skip loading it
		fmt.Printf("no data to load from database\n")
		return nil
	}

	wg := sync.WaitGroup{}

	helper := func(i int, block []byte) {
		fmt.Printf("starting to process block %v after %v\n", i, time.Now().Sub(startDate))

		helperStartDate := time.Now()

		var theseNodes []DirFullInfo
		err := json.Unmarshal(block, &theseNodes)
		if err != nil {
			panic(fmt.Errorf("error reading block %v: %v", i, err))
		}
		unmarshalDate := time.Now()
		fmt.Printf("unmarshaled block %v in %v\n", i, unmarshalDate.Sub(helperStartDate))
		//tempMap := NewPathMap(fmt.Sprintf("block %v", i))
		tempMap := NewPathMap()
		//tempMap := NewCacheFor(&F.nodes)
		//count := 0
		//cacheResponses := make([]StattedDir, len(theseNodes))
		//statResponses := make([]StattedDir, len(theseNodes))
		stats := make([]DirStats, len(theseNodes))
		for i, node := range theseNodes {
			// check the file system for an updated timestamp
			//updated := F.statDirSync(node.P)
			stats[i] = f.statDirSync(node.P)

		}
		for i, node := range theseNodes {
			updated := stats[i]
			// save the cached value
			container := tempMap.GetLeaf(node.P, true, false)
			container.mapNode = mapNode{DirStats: node.DirStats, Files: node.F}

			// check the contents of the directory if our info might be out-of-date
			if updated.T != container.T {
				container.T = updated.T
				f.listDirAsync(node.P)
			}
		}
		// merge our temp map back into the main map
		tempMap.recomputeNumDescendentsRecursive()
		f.nodes.mergeIn(tempMap)
		fmt.Printf("statted inodes of block %v in %v\n", i, time.Now().Sub(unmarshalDate))

		wg.Done()

	}

	bufferedReader := bufio.NewReader(reader)

	i := 0

	for {

		nextBlock, err := bufferedReader.ReadBytes(byte('\n'))
		if err != nil && err != io.EOF {
			return err
		}
		wg.Add(1)
		go helper(i, nextBlock)
		i++
		if err == io.EOF {
			break
		}
	}
	loadDate := time.Now()
	fmt.Printf("read %s in %v\n", dbPath, loadDate.Sub(startDate))
	fmt.Printf("waiting for all db-loading goroutines to complete\n")
	wg.Wait()

	// can't start polling until after we've loaded everything from cache
	//F.goPoll()

	fmt.Printf("loaded db and statted its contents in %v\n", time.Now().Sub(loadDate))
	if err != nil {
		return err
	}

	return nil
}

/*func (F *Finder) putNode(node StattedDir) {
	//fmt.Printf("putting date of %v for %s\n", node.content.DirStats.T, node.content.P)
	F.nodes[node.P] = node
}*/

func (f *Finder) entries() []DirFullInfo {
	nodes := make([]DirFullInfo, 0)
	for _, node := range f.nodes.DumpAll() {
		if node.T != 0 {
			nodes = append(nodes, node)
		}
	}
	less := func(i int, j int) bool {
		return nodes[i].P < nodes[j].P
	}
	sort.Slice(nodes, less)

	return nodes
}

func (f *Finder) dumpDb() error {
	startDate := time.Now()
	//dbPath := F.dbPath
	fmt.Printf("dumping db\n")

	tempPath := f.dbPath + ".tmp"

	// sort dirs
	var entryList = f.entries()

	// dump to output file

	numBlocks := 100

	blockSize := len(entryList)/numBlocks + 1

	bytes := []byte{}
	var err error

	blockMin := 0

	for {
		if blockMin >= len(entryList) {
			break
		}

		blockMax := blockMin + blockSize
		if blockMax > len(entryList) {
			blockMax = len(entryList)
		}

		if blockMax > len(entryList) {
			blockMax = len(entryList)
		}
		block := entryList[blockMin:blockMax]
		// dump block to tempPath
		byteBlock, err := json.Marshal(block)
		if err != nil {
			return err
		}
		if blockMin != 0 {
			bytes = append(bytes, byte('\n'))
		}
		bytes = append(bytes, byteBlock...)

		blockMin = blockMax
	}
	err = ioutil.WriteFile(tempPath, bytes, 0777)
	if err != nil {
		return err
	}

	// move from tempPath to F.dbPath
	err = os.Rename(tempPath, f.dbPath)
	if err != nil {
		return err
	}
	fmt.Printf("dumped db in %v\n", time.Now().Sub(startDate))
	return nil
}

func (f *Finder) Shutdown() {
	fmt.Printf("shutting down\n")
	f.WaitUntilIdle()
	if f.wasModified() {
		f.dumpDb()
	} else {
		fmt.Printf("Skipping dumping unmodified db\n")
	}
}

func New(filterDir func(items *ChildList), dbPath string) *Finder {
	finder := &Finder{

		filterDir: filterDir,

		statResponses:      make(chan *statResponse, 100),
		childListResponses: make(chan *ChildList, 100),

		//nodes:  *NewPathMap("primary"),
		nodes:  *NewPathMap(),
		dbPath: dbPath,

		requests: sync.WaitGroup{},
	}

	//finder.goPoll()
	err := finder.loadDb()
	if err != nil {
		panic(err)
	}
	fmt.Printf("done parsing db\n")
	finder.goPoll()
	//finder.WaitUntilIdle()
	//fmt.Printf("done loading db\n")
	return finder
}

func (f *Finder) goPoll() {
	for i := 0; i < 4; i++ {
		go func() {
			// make a cache of map keys to avoid having to often lock the maps near the root
			cache := NewCacheFor(&f.nodes)
			modified := false
			for {
				// check for more work
				select {
				case stat := <-f.statResponses:
					// we got a response telling the stats of several directories
					if stat == nil {
						return
					}
					for _, node := range stat.dirs {
						oldNode := cache.tryGet(node.P, true)
						if oldNode != nil && oldNode.T != 0 && oldNode.T == node.DirStats.ModTime() {
							// no change in modification time, so our cached info is already correct

							// we already spawned goroutines to check modification times and don't need to spawn more
						} else {
							newNode := cache.getOrCreate(node.P, true)
							fmt.Printf("entries was out-of-date for %s\n", node.P)
							if oldNode != nil {
								fmt.Printf("prev modtime of %s was %v, new modtime is %v\n", node.P, oldNode.ModTime(), node.ModTime())
							} else {
								fmt.Printf("entries contained no entry for %s\n", node.P)
							}
							newNode.mapNode = mapNode{DirStats: node.DirStats, Files: []string{}}
							// modification time was updated, so scan for child directories
							if node.DirStats.ModTime() != 0 {
								f.listDirAsync(node.P)
							}
							if !modified {
								modified = true
								f.setModified(true)
							}
						}
						//F.numServicedRequests++
						//fmt.Printf("statted %v\n", node.P)
					}
					f.requests.Done()

				case children := <-f.childListResponses:
					// we got a response telling the subdirectories of several directories
					if children == nil {
						return
					}
					node := cache.getOrCreate(children.Path, true)
					// record any discovered Files
					node.Files = children.Files

					// stat any children too
					for _, child := range children.SubDirs {
						f.statDirAsync(children.Path + "/" + child)
					}
					if !modified {
						modified = true
						f.setModified(true)
					}

					f.requests.Done()
				}
			}
		}()

	}
}

// goPoll really only exists to synchronize access to the maps in the Finder
// we could implement a concurrent hashmap for more concurrency if it becomes important
/*func (F *Finder) goPoll() {
	go func() {
		for {
			select {
			case stat := <-F.statResponses:
				if stat == nil {
					return
				}
				for _, node := range stat.dirs {
					oldstat, ok := F.nodes[node.P]
					if stat.fromCache {
						F.putNode(node)
					} else {
						if ok && node.DirStats.ModTime() != 0 && oldstat.DirStats.ModTime() == node.DirStats.ModTime() {
							// no change in modification time, so our cached info is already correct

							// we already spawned goroutines to check modification times and don't need to spawn more
						} else {
							F.putNode(node)
							fmt.Printf("entries was out-of-date for %s\n", node.P)
							if ok {
								fmt.Printf("prev modtime of %s was %v, new modtime is %v\n", node.P, oldstat.DirStats.ModTime(), node.DirStats.ModTime())
							} else {
								fmt.Printf("entries contained no entry for %s\n", node.P)
							}
							F.modified = true
							// modification time was updated, so scan for child directories
							if node.DirStats.ModTime() != 0 {
								F.listDirAsync(node.P)
							}
						}
						//F.numServicedRequests++
					}
				}
				if !stat.fromCache {
					F.requests.Done()
				}

			case children := <-F.childListResponses:
				if children == nil {
					return
				}
				F.children[children.Path] = childrenSnapshot{content: children}
				// stat any children too
				for _, child := range children.SubDirs {
					F.statDirAsync(child)
				}
				//F.numServicedRequests++
				F.requests.Done()
			}
		}
	}()
}*/

func (f *Finder) StartFind(path string) {
	f.statDirAsync(path)
}

func (f *Finder) statDirAsync(path string) {
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	f.requests.Add(1)
	//fmt.Printf("statDirAsync %v\n", Path)

	go func() {
		//fmt.Printf("statDirAsync goroutine before statDirSync %v\n", Path)
		stats := f.statDirSync(path)
		//fmt.Printf("statDirAsync goroutine after statDirSync %v\n", Path)

		f.statResponses <- &statResponse{dirs: []StattedDir{StattedDir{P: path, DirStats: stats}}}
	}()
}

func (f *Finder) statDirSync(path string) DirStats {

	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	fileInfo, err := os.Stat(path)
	// in case of any error, pass an empty response into the channel

	var stats DirStats
	if err == nil {
		stats = DirStats{T: fileInfo.ModTime().UnixNano()}
	}

	return stats
}

func (f *Finder) listDirAsync(path string) {
	if path == "/usr/local/google/workspace/aosp" {
		fmt.Printf("listing %v\n", path)
	}
	f.requests.Add(1)
	//fmt.Printf("listDirAsync")
	go func() {
		//fmt.Printf("listDirAsync goroutine")
		children, _ := ioutil.ReadDir(path)
		// in case of any error, pass an empty response into the channel

		var subdirs []string
		var subfiles []string

		for _, child := range children {
			linkBits := child.Mode() & os.ModeSymlink
			isLink := (linkBits != 0)
			if child.IsDir() {
				if !isLink {
					// Skip symlink dirs.
					// We don't have to support symlink dirs because that would cause duplicates
					subdirs = append(subdirs, child.Name())
				}
			} else {
				// We do have to support symlink files because the link name might be different
				// than the target name
				// (for example, Android.bp -> build/soong/root.bp)
				subfiles = append(subfiles, child.Name())
			}

		}
		response := &ChildList{Path: path, SubDirs: subdirs, Files: subfiles}
		f.filterDir(response)
		f.childListResponses <- response
	}()
}

func (f *Finder) WaitUntilIdle() {
	startDate := time.Now()
	fmt.Printf("waiting for pending requests to complete\n")
	f.requests.Wait()
	fmt.Printf("Is idle after %v\n", time.Now().Sub(startDate))
}

func (f *Finder) HasFreshData(path string) bool {
	node := f.nodes.GetLeaf(path, false, false)
	if node == nil {
		return false
	}
	if node.T == 0 {
		return false
	}
	return true
}

func (f *Finder) prepareToFind(path string) {
	// this method could be implemented with more granularity but it's probably still fine

	fmt.Printf("Find waiting for finder to be idle\n")
	f.WaitUntilIdle()

	if !f.HasFreshData(path) {
		fmt.Printf("Find requesting load of %s\n", path)
		f.StartFind(path)
		fmt.Printf("Find waiting for load of %s\n", path)
		f.WaitUntilIdle()
	}
}

func (f *Finder) FindNamed(rootPath string, fileName string) []string {
	scanStart := time.Now()
	fmt.Printf("finder finding %v using cache\n", rootPath)

	f.prepareToFind(rootPath)

	matching := func(filePath string) bool {
		_, leaf := filepath.Split(filePath)
		return leaf == fileName
	}
	channel := f.findInCacheMultithreaded(rootPath, matching, 100)
	results := <-channel
	fmt.Printf("found %v files under %v in %v using cache\n", len(results), rootPath, time.Now().Sub(scanStart))

	return results
}

func (f *Finder) findInCacheMultithreaded(path string, predicate func(string) bool, numThreads int) chan []string {
	//fmt.Printf("scanning %v with about %v threads\n", path, numThreads)
	if numThreads < 2 {
		return f.findInCacheAsync(path, predicate)
	}
	//scanStart := time.Now()

	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	channel := make(chan []string, 1)

	go func() {
		node := f.nodes.GetLeaf(path, false, false)
		totalWork := 0
		for _, child := range node.children {
			totalWork += child.approximateNumDescendents
		}
		channels := make([]chan []string, 0)
		results := f.listMatching(path, predicate, node)

		// spawn threads to process child directories
		for key, child := range node.children {
			numChildThreads := numThreads * child.approximateNumDescendents / totalWork
			channels = append(channels, f.findInCacheMultithreaded(path+"/"+key, predicate, numChildThreads))
		}
		// collect results
		for _, childChannel := range channels {
			childResults := <-childChannel
			results = append(results, childResults...)
		}
		//fmt.Printf("found %v files under sub-path %v in %v using cache\n", len(results), path, time.Now().Sub(scanStart))

		channel <- results
		close(channel)
	}()

	return channel
}
func (f *Finder) findInCacheAsync(path string, predicate func(string) bool) chan []string {
	channel := make(chan []string, 1)
	go func() {
		results := f.findInCacheSync(path, predicate)
		channel <- results
		close(channel)
	}()
	return channel
}

func (f *Finder) findInCacheSync(path string, predicate func(string) bool) []string {
	//scanStart := time.Now()

	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	//fmt.Printf("finder finding %v using cache\n", path)
	cache := NewCacheFor(&f.nodes)

	searchDirs := []string{path}
	startingMap := cache.tryGet(path, false)
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
	//fmt.Printf("found %v files synchronously under %v in %v using cache\n", len(matches), path, time.Now().Sub(scanStart))
	return matches
}

func (f *Finder) listMatching(path string, predicate func(string) bool, node *pathMap) []string {
	matches := []string{}
	for _, fileName := range node.Files {
		filePath := path + "/" + fileName
		if predicate(filePath) {
			matches = append(matches, filePath)
		}
	}
	return matches
}

// a mapCache caches the nodes of a pathMap to make multithreaded access faster
type mapCache struct {
	entries map[string]*pathMap
}

func NewCacheFor(m *pathMap) *mapCache {
	cache := mapCache{entries: make(map[string]*pathMap, 8)}
	cache.entries[""] = m
	return &cache
}

func (c *mapCache) getOrCreate(path string, threadsafe bool) *pathMap {
	return c.get(path, true, threadsafe)
}

func (c *mapCache) tryGet(path string, threadsafe bool) *pathMap {
	return c.get(path, false, threadsafe)
}

func (c *mapCache) get(path string, createIfNotFound bool, threadsafe bool) *pathMap {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	//if len(Path) < 1 {
	//	panic("illegal empty Path")
	//}
	cached, found := c.entries[path]
	if found {
		// if the result is already known, then return it
		return cached
	} else {
		// if the result is not already known, then walk up the tree to find a node that is known
		dir, leaf := filepath.Split(path)
		if dir == path {
			panic(fmt.Sprintf("returned Path %v as parent of itself, %v\n", dir, path))
		}
		//fmt.Printf("getting parent map for %s\n", Path)
		//if len(dir) == 0 {
		//	panic(fmt.Sprintf("illegal empty Path returned as parent of %v\n", Path))
		//}
		parentMap := c.get(dir, createIfNotFound, threadsafe)
		if parentMap == nil {
			// not found and caller doesn't wish to create, so skip it
			return nil
		}
		return parentMap.GetLeaf(leaf, createIfNotFound, threadsafe)
	}
}

// a pathMap implements the tree structure of nodes and supports threadsafe access
type pathMap struct {
	//DirFullInfo
	mapNode

	children map[string]*pathMap
	mutex    sync.Mutex

	approximateNumDescendents int

	//Threadsafe bool
	//Name string
}

func NewPathMap() *pathMap {
	result := &pathMap{children: make(map[string]*pathMap, 4), approximateNumDescendents: 1}
	//result.P = "/"
	//result.Name = name
	return result
}

func (m *pathMap) lock() {
	m.mutex.Lock()
}

func (m *pathMap) unlock() {
	m.mutex.Unlock()
}

func (m *pathMap) updateNumDescendents() int {
	count := 1
	for _, child := range m.children {
		count += child.approximateNumDescendents
	}
	m.approximateNumDescendents = count
	return count
}

func (m *pathMap) recomputeNumDescendentsRecursive() {
	for _, child := range m.children {
		child.recomputeNumDescendentsRecursive()
	}
	m.updateNumDescendents()
}

func (m *pathMap) pathComponents(path string) (components []string) {
	//fmt.Printf("getting components of %v\n", Path)
	if len(path) == 0 {
		return []string{}
	}
	if path == "/" {
		return []string{}
	}
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	dir, leaf := filepath.Split(path)
	return append(m.pathComponents(dir), leaf)
}

func (m *pathMap) GetLeaf(path string, createIfNotFound bool, threadsafe bool) *pathMap {
	components := m.pathComponents(path)
	return m.getLeafForComponents(components, createIfNotFound, threadsafe)
}

func (m *pathMap) getLeafForComponents(components []string, createIfNotFound bool, threadsafe bool) *pathMap {
	if len(components) == 0 {
		return m
	}

	childComponent := components[0]
	if len(childComponent) < 1 {
		panic(fmt.Sprintf("illegal component '%v' from components %v", childComponent, components))
	}

	if threadsafe {
		m.lock()
		defer m.unlock()
	}

	//fmt.Printf("trying to get leaf at %v for Path components %v with createIfNotFound = %v\n", m.Name, components, createIfNotFound)

	subMap, found := m.children[childComponent]

	if !found {
		if createIfNotFound {
			subMap = m.newChild(childComponent)
			m.children[childComponent] = subMap
		} else {
			return nil
		}
	}
	result := subMap.getLeafForComponents(components[1:], createIfNotFound, threadsafe)

	return result

}

func (m *pathMap) newChild(name string) *pathMap {
	//child := NewPathMap(fmt.Sprintf("%v/%v", m.Name, name))
	child := NewPathMap()
	//child.P = filepath.Join(m.P, name)
	//fmt.Printf("Creating child %v\n", child.P)
	return child
}

// this method isn't threadsafe but we don't need it to be
func (m *pathMap) DumpAll() []DirFullInfo {
	results := []DirFullInfo{}
	m.dumpInto("", &results)
	return results
}

// this method isn't threadsafe but we don't need it to be
func (m *pathMap) dumpInto(path string, results *[]DirFullInfo) {
	*results = append(*results, DirFullInfo{StattedDir{path, m.DirStats}, m.Files})
	for key, child := range m.children {
		childPath := path + "/" + key
		//childPath := filepath.Join(Path, key)
		child.dumpInto(childPath, results)
	}
}

func (m *pathMap) mergeIn(other *pathMap) {
	//if !m.Threadsafe {
	//	panic(fmt.Sprintf("called mergeIn on non-threadsafe map %v\n", m.Name))
	//}
	m.lock()
	defer m.unlock()
	//fmt.Printf("merging other %v into this %v\n", other.P, m.P)

	for key, theirs := range other.children {
		ours, found := m.children[key]
		if found {
			//fmt.Printf("recursing into children for merge\n")
			ours.mergeIn(theirs)
		} else {
			//fmt.Printf("copying key %v from other for Path %v\n", key, m.P)
			m.children[key] = theirs
			//m.children[key].Name = "(" + m.Name + ")" + theirs.Name
		}
	}
	if other.T != 0 {
		m.mapNode = other.mapNode
	}
	m.updateNumDescendents()
}
