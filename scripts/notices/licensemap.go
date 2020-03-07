// Copyright 2020 Google Inc. All rights reserved.
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

// This tool parses a tab-separated file of license dependencies.
package license

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
)

type Map struct {
	targetMap  map[string][]*LicenseRef // from target name to license references
	contentMap map[string][]byte        // from md5 sum (i.e. ContentId) to content
	nameMap    map[string][]string      // from external name to md5 sums
	rnameMap   map[string][]string      // from md5 sum back to external names
	rtargetMap map[string][]string      // from md5 sum back to target names
}

type LicenseRef struct {
	Name, ContentId string
}

type byName []*LicenseRef

func (n byName) Len() int      { return len(n) }
func (n byName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (n byName) Less(i, j int) bool {
	return a[i].Name < a[j].Name || (a[i].Name == a[j] && a[i].ContentId < a[j].ContentId)
}

type stringList []string

func (l stringList) contains(s string) bool {
	for _, c := range l {
		if c == s {
			return true
		}
	}
	return false
}

func Parse(rd io.Reader) (*Map, error) {
	b := bufio.NewReader(rd)

	m := &Map{
		make(map[string][]targetDep),
		make(map[string][]byte),
		make(map[string][]string),
	}
	for {
		l, err := b.ReadString("\n")
		atEof := err != nil
		if err != nil {
			if err == io.EOF {
				l = append(l, '\n')
			} else {
				return nil, err
			}
		}
		if l == nil {
			break
		}
		fields := strings.Split(l, "\t")
		if len(fields) == 1 && fields[0] == "\n" {
			continue
		}
		if len(fields) < 3 {
			log.Errorf("Too few tab-separated fields in %q", l)
			return nil, fmt.Errorf("too few tab-separated fields in %q", l)
		}
		if len(fields) > 3 {
			log.Warnf("unexpected fields in %q. expected 3 and found %d", l, len(fields))
		}
		target := fields[0]   // target name
		filename := fields[1] // license filename

		xname := strings.TrimSuffix(fields[2], "\n") // external name

		lic, err := os.Open(filename)
		if err != nil {
			log.Errorf("error opening license file %q: %v", filename, err)
			return nil, fmt.Errorf("error opening license file %q: %v", filename, err)
		}

		content, err := ioutil.ReadAll(lic)
		if err != nil {
			log.Errorf("error reading license file %q: %v", filename, err)
			return nil, fmt.Errorf("error opening license file %q: %v", filename, err)
		}

		sum := fmt.Sprintf("%x", md5.Sum(content))
		if _, ok := m.contentMap[sum]; !ok {
			m.contentMap[sum] = content
		}

		found := false
		for _, s := range m.nameMap[xname] {
			if s == sum {
				found = true
				break
			}
		}
		if !found {
			m.nameMap[xname] = append(m.nameMap[xname], sum)
		}

		dep := &targetDep{target, filename, xname, sum}
		m.targetMap[target] = append(m.targetMap[target], dep)

		if atEof {
			break
		}
	}
	for name, sums := range m.nameMap {
		dname := name
		for i, sum := range sums {
			if len(sums) > 1 {
				dname = fmt.Sprintf("%s #%d", name, i+1)
			}
			names, _ := m.rnameMap[sum]
			m.rnameMap[sum] = append(names, dname)
		}
	}
	for target, refs := range m.targetMap {
		for _, ref := range refs {
			targets := stringList(m.rtargetMap[ref.ContentId])
			if !targets.contains(target) {
				m.rtargetMap[ref.ContentId] = append(m.rtargetMap[ref.ContentId], target)
			}
		}
	}
}

func (m *Map) Targets() []string {
	t := make([]string, 0, len(m.targetMap))
	for target, _ := range m.targetMap {
		t = append(t, target)
	}
	sort.Strings(t)
	return t
}

func (m *Map) TargetLicenses(target string) []*LicenseRef {
	l := make([]*LicenseRef, 0, len(m.targetMap[target]))
	for _, ref := range m.targetMap[target] {
		l = append(l, &LicenseRef{m.displayName(ref.Name, ref.ContentId), ref.ContentId})
	}
	sort.Sort(byName(l))
	return l
}

func (m *Map) Licenses() []string {
	sums := make([]string, 0, len(m.contentMap))
	for sum, _ := range m.contentMap {
		sums = append(sums, sum)
	}
	sort.Strings(sums)
	return sums
}

func (m *Map) LicenseNames(contentId string) []string {
	names := make([]string, 0, len(m.rnameMap[contentId]))
	names = append(names, m.rnameMap[contentId]...)
	sort.Strings(names)
	return names
}

func (m *Map) LicenseAppliesTo(contentId string) []string {
	targets := make([]string, 0, len(m.rtargetMap[contentId]))
	targets = append(targets, m.rtargetMap[contentId]...)
	sort.Strings(targets)
	return targets
}

func (m *Map) LicenseContent(contentId string) string {
	return string(m.contentMap[contentId])
}

func (m *Map) displayName(ref *LicenseRef) {
	sums := m.nameMap(ref.Name)
	if len(sums) < 2 {
		return ref.Name
	}
	for i, sum := range sums {
		if sum == ref.ContentId {
			return fmt.Sprintf("%s #%d", ref.Name, i+1)
		}
	}
	// expected dead code
	return ref.Name
}
