// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"

	"android/soong/compliance/license_metadata_proto"

	"google.golang.org/protobuf/encoding/prototext"
)

var (
	// ConcurrentReaders is the size of the task pool for limiting resource usage e.g. open files.
	ConcurrentReaders = 5
)

// LicenseFile
type LicenseFile struct {
	lm   *licenseMetadataImp
	file string
}

func (lf LicenseFile) GetProjects() []string {
	return append([]string{}, lf.lm.metadata[lf.file].Projects...)
}

// LicenseSet
type LicenseSet struct {
	lm    *licenseMetadataImp
	files map[string]interface{}
}

func (ls LicenseSet) Add(other LicenseSet) {
	if ls.lm != other.lm {
		panic(fmt.Errorf("attempt to Add license sets from differen metadata"))
	}
	for f := range other.files {
		ls.files[f] = nil
	}
}

func (ls LicenseSet) Files() []LicenseFile {
	var files []LicenseFile
	for f := range ls.files {
		files = append(files, LicenseFile{ls.lm, f})
	}
	return files
}

func (ls LicenseSet) Projects() []string {
	pset := make(map[string]interface{})
	for f := range ls.files {
		for _, p := range ls.lm.metadata[f].Projects {
			pset[p] = nil
		}
	}
	projects := make([]string, 0, len(pset))
	for p := range pset {
		projects = append(projects, p)
	}
	sort.Strings(projects)
	return projects
}

// LicenseMetadata describes the license metadata for a set of root targets.
type LicenseMetadata interface {
	// GetProjects returns the projects associated with the metadata file `f`
	GetProjects(f LicenseFile) ([]string, error)
	// WalkRestricted returns the set of metadata files to treat as 'restricted'.
	WalkRestricted() LicenseSet
	// WalkDepsForCondition returns the set of metadata with license `condition`.
	WalkDepsForCondition(condition string) LicenseSet
}

// licenseMetadataImp implements the LicenseMetadata interface.
type licenseMetadataImp struct {
	// rootFiles identifies the original set of files to read (immutable)
	rootFiles []string
	// metadata identifies the requested files (initially nil) and the parsed results (guarded by recv.mu)
	metadata map[string]*metadataFile
}

// GetProjects returns the projects associated with the metadata file `f`
func (lm *licenseMetadataImp) GetProjects(f LicenseFile) ([]string, error) {
	mf, ok := lm.metadata[f.file]
	if !ok {
		return nil, fmt.Errorf("invalid metadata file %q", f)
	}
	return mf.Projects, nil
}

func (lm *licenseMetadataImp) WalkRestricted() LicenseSet {
	rmap := make(map[string]interface{})
	cmap := make(map[string]interface{})

	var walkContainer, walkNonContainer func(string)

	walkNonContainer = func(f string) {
		rmap[f] = nil
		for _, d := range lm.metadata[f].Deps {
			if _, alreadyWalked := rmap[d]; alreadyWalked {
				if _, asContainer := cmap[d]; asContainer {
					delete(cmap, d)
					walkNonContainer(d)
				}
				continue
			}
			walkNonContainer(d)
		}
	}

	walkContainer = func(f string) {
		rmap[f] = nil
		cmap[f] = nil
		for _, d := range lm.metadata[f].Deps {
			if !lm.metadata[d].isRestricted {
				continue
			}
			if _, alreadyWalked := rmap[d]; alreadyWalked {
				continue
			}
			if lm.metadata[d].GetIsContainer() {
				walkContainer(d)
			} else {
				walkNonContainer(d)
			}
		}
	}

	for _, r := range lm.rootFiles {
		if !lm.metadata[r].isRestricted {
			continue
		}
		if _, alreadyWalked := rmap[r]; alreadyWalked {
			continue
		}
		if lm.metadata[r].GetIsContainer() {
			walkContainer(r)
		} else {
			walkNonContainer(r)
		}
	}

	return LicenseSet{lm, rmap}
}

func (lm *licenseMetadataImp) WalkDepsForCondition(condition string) LicenseSet {
	if strings.HasPrefix(condition, "restrict") {
		return lm.WalkRestricted()
	}

	matched := make(map[string]interface{})
	wmap := make(map[string]interface{})

	var walk func(string)

	walk = func(f string) {
		wmap[f] = nil
		for _, c := range lm.metadata[f].LicenseConditions {
			if c == condition {
				matched[f] = nil
				break
			}
		}
		for _, d := range lm.metadata[f].Deps {
			if _, alreadyWalked := wmap[d]; alreadyWalked {
				continue
			}
			walk(d)
		}
	}

	for _, r := range lm.rootFiles {
		if _, alreadyWalked := wmap[r]; alreadyWalked {
			continue
		}
		walk(r)
	}

	return LicenseSet{lm, matched}
}

// newLicenseMetadataImp constructs a new instance of licenseMetadataImp.
func newLicenseMetadataImp() *licenseMetadataImp {
	return &licenseMetadataImp{
		[]string{},
		make(map[string]*metadataFile),
	}
}

type metadataFile struct {
	license_metadata_proto.LicenseMetadata

	// isRestricted is true when `LicenseConditions` for this or any dependency contains restricted conditon.
	isRestricted bool
	// isProprietary is true when `LicenseConditions` for this or any dependency contains proprietary condition.
	isProprietary bool
	// effectiveLicenseTexts lists the transitive closure of license texts including dependencies.
	effectiveLicenseTexts []string
}

// result describes the outcome of reading and parsing a single license metadata file.
type result struct {
	// file identifies the path to the license metadata file
	file string
	// meta contains the parsed metadata or nil if an error
	meta *metadataFile
	// err is nil unless an error occurs
	err error
}

// receiver coordinates the tasks for reading and parsing license metadata files.
type receiver struct {
	// ls accumulates the read metadata (guarded by mu)
	lm *licenseMetadataImp
	// task provides a fixed-size task pool to limit concurrent open files etc.
	task chan bool
	// results returns one metadata file result at a time
	results chan *result
	// wg detects when done
	wg sync.WaitGroup
	// mu guards ls against concurrent update
	mu sync.Mutex
}

// ReadLicenseMetadata reads and parses `files` and their dependencies into a LicenseMetadata.
func ReadLicenseMetadata(files []string) (LicenseMetadata, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no license metadata to analyze")
	}

	lm := newLicenseMetadataImp()
	for _, f := range files {
		if strings.HasSuffix(f, ".meta_lic") {
			lm.rootFiles = append(lm.rootFiles, f)
		} else {
			lm.rootFiles = append(lm.rootFiles, f+".meta_lic")
		}
	}

	recv := &receiver{lm: lm, task: make(chan bool, ConcurrentReaders), results: make(chan *result, ConcurrentReaders), wg: sync.WaitGroup{}}
	for i := 0; i < ConcurrentReaders; i++ {
		recv.task <- true
	}

	readFiles := func() {
		// identify the metadata files to schedule reading tasks for
		for _, f := range lm.rootFiles {
			lm.metadata[f] = nil
		}

		// schedule tasks to read the files
		for _, f := range lm.rootFiles {
			recv.wg.Add(1)
			<-recv.task
			go readFile(recv, f)
		}

		// schedule a task to wait until finished and close the channelm.
		go func() {
			recv.wg.Wait()
			close(recv.task)
			close(recv.results)
		}()
	}
	go readFiles()

	// tasks to read license metadata files are scheduled; read and process results from channel
	var err error
	for recv.results != nil {
		select {
		case r, ok := <-recv.results:
			if ok {
				// handle errors by nil'ing ls, setting err, and clobbering results channel
				if r.err != nil {
					err = r.err
					lm = nil
					recv.results = nil
					continue
				}

				// record the parsed metadata (guarded by mutex)
				recv.mu.Lock()
				recv.lm.metadata[r.file] = r.meta
				recv.mu.Unlock()
			} else {
				// finished -- nil the results channel
				recv.results = nil
			}
		}
	}

	// single task has reference to `ls` at this point
	if lm != nil {
		// reading and parsing succeeded -- calculate effective conditions and license texts
		for f := range lm.metadata {
			lm.metadata[f].isRestricted = isRestricted(lm.metadata[f], lm)
			lm.metadata[f].isProprietary = isProprietary(lm.metadata[f], lm)
			flattenTexts(lm.metadata[f], lm)
		}
	}
	return lm, err
}

// readFile is a task to read and parse a single license metadata file, and to schedule
// additional tasks for reading and parsing dependencies as necessary.
func readFile(recv *receiver, file string) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		recv.results <- &result{file, nil, err}
		fmt.Fprintf(os.Stderr, "error reading license metadata %q: %w", file, err)
		return
	}

	mf := metadataFile{}

	err = prototext.Unmarshal(data, &mf.LicenseMetadata)
	if err != nil {
		recv.results <- &result{file, nil, err}
		fmt.Fprintf(os.Stderr, "error license metdata %q: %w", file, err)
		return
	}

	// send result for this file and release task before scheduling dependencies,
	// but do not signal done to WaitGroup until dependencies are scheduled.
	recv.results <- &result{file, &mf, nil}
	recv.task <- true

	// schedule tasks as necessary to read dependencies
	for _, d := range mf.Deps {
		// decide and record whether to schedul task in critical section
		recv.mu.Lock()
		_, alreadyScheduled := recv.lm.metadata[d]
		if !alreadyScheduled {
			recv.lm.metadata[d] = nil
		}
		recv.mu.Unlock()
		// schedule task to read dependency file outside critical section
		if !alreadyScheduled {
			recv.wg.Add(1)
			<-recv.task
			go readFile(recv, d)
		}
	}

	// signal done after scheduling dependencies
	recv.wg.Done()
}

// isRestricted returns true if `mf` or any dependency has 'restricted' condition.
func isRestricted(mf *metadataFile, lm *licenseMetadataImp) bool {
	// check whether already has "restricted"
	for _, c := range mf.LicenseConditions {
		if strings.HasPrefix(c, "restrict") {
			return true
		}
	}

	// add "restricted" if any dependency has "restricted"
	for _, d := range mf.Deps {
		if isRestricted(lm.metadata[d], lm) {
			return true
		}
	}

	return false
}

// isProprietary returns true if `mf` or any dependency has 'proprietary' condition.
func isProprietary(mf *metadataFile, lm *licenseMetadataImp) bool {
	// check whether already has "proprietary"
	for _, c := range mf.LicenseConditions {
		if c == "proprietary" {
			return true
		}
	}

	// add "proprietary" if any dependency has "proprietary"
	for _, d := range mf.Deps {
		if isProprietary(lm.metadata[d], lm) {
			return true
		}
	}

	return false
}

// flattenTexts calculates the transitive closure of license texts for `mf` from its dependencies.
func flattenTexts(mf *metadataFile, lm *licenseMetadataImp) {
	// copy license texts
	mf.effectiveLicenseTexts = append(mf.effectiveLicenseTexts, mf.LicenseTexts...)

	// indicate license texts already recorded
	recorded := make(map[string]bool)
	for _, t := range mf.effectiveLicenseTexts {
		recorded[t] = true
	}

	// add unrecorded license texts from dependencies
	for _, d := range mf.Deps {
		for _, t := range lm.metadata[d].LicenseTexts {
			if _, alreadyRecorded := recorded[t]; !alreadyRecorded {
				mf.effectiveLicenseTexts = append(mf.effectiveLicenseTexts, t)
				recorded[t] = true
			}
		}
	}
}
