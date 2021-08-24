// Copyright (C) 2021 The Android Open Source Project
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

package sdk

import (
	"fmt"
	"sort"
	"strings"

	"android/soong/android"
)

// Supports customizing sdk snapshot output based on target build release.

// buildRelease represents the version of a build system used to create a specific release.
//
// The name of the release, is the same as the code for the dessert release, e.g. Q, R, S, T, etc.
type buildRelease struct {
	// The name of the release, e.g. Q, R, S, T, etc.
	name string

	// The index of this structure within the buildReleases list.
	ordinal int
}

// String returns the name of the build release.
func (s *buildRelease) String() string {
	return s.name
}

// buildReleaseSet represents a set of buildRelease objects.
type buildReleaseSet struct {
	contents map[string]*buildRelease
}

// addItem adds a build release to the set.
func (s *buildReleaseSet) addItem(release *buildRelease) {
	s.contents[release.name] = release
}

// addRange adds all the build releases from start (inclusive) to end (inclusive).
func (s *buildReleaseSet) addRange(start *buildRelease, end *buildRelease) {
	for i := start.ordinal; i <= end.ordinal; i += 1 {
		s.addItem(buildReleases[i])
	}
}

// contains returns true if the set contains the specified build release.
func (s *buildReleaseSet) contains(release *buildRelease) bool {
	x, ok := s.contents[release.name]
	return ok && x == release
}

// String returns a string representation of the set, sorted by build release name.
func (s *buildReleaseSet) String() string {
	list := []string{}
	for name, _ := range s.contents {
		list = append(list, name)
	}
	sort.Strings(list)
	return fmt.Sprintf("%s", list)
}

var (
	// nameToBuildRelease contains a map from name to build release.
	nameToBuildRelease = map[string]*buildRelease{}

	// buildReleases lists all the available build releases.
	buildReleases = []*buildRelease{}

	// Add the build releases from oldest to newest.
	buildReleaseQ = initBuildRelease("Q")
	buildReleaseR = initBuildRelease("R")
	buildReleaseS = initBuildRelease("S")
	buildReleaseT = initBuildRelease("T")

	// The current build release is the last one added.
	currentBuildRelease = buildReleases[len(buildReleases)-1]
)

// initBuildRelease creates a new build release with the specified name.
func initBuildRelease(name string) *buildRelease {
	ordinal := len(nameToBuildRelease)
	release := &buildRelease{name: name, ordinal: ordinal}
	nameToBuildRelease[name] = release
	buildReleases = append(buildReleases, release)
	return release
}

// nameToRelease maps from build release name to the corresponding build release (if it exists) or
// the error if it does not.
func nameToRelease(name string) (*buildRelease, error) {
	if r, ok := nameToBuildRelease[name]; ok {
		return r, nil
	}

	availableReleases := strings.Join(android.SortedStringKeys(nameToBuildRelease), ", ")
	return nil, fmt.Errorf("unknown release %q, expected one of %s", name, availableReleases)
}

// parseBuildReleaseSet parses a build release set string specification into a build release set.
//
// The specification consists of a number of comma separated pieces. Each piece can be one of the
// following:
// * a single build release name, e.g. Q, R, etc.
// * a closed range (inclusive to inclusive), e.g. Q-S
// * an open range, e.g. T+.
//
// This returns the set if the specification was valid or an error.
func parseBuildReleaseSet(s string) (*buildReleaseSet, error) {
	set := &buildReleaseSet{contents: map[string]*buildRelease{}}

	pieces := strings.Split(s, ",")
	for _, piece := range pieces {
		if strings.HasSuffix(piece, "+") {
			piece = strings.TrimSuffix(piece, "+")
			start, err := nameToRelease(piece)
			if err != nil {
				return nil, err
			}
			end := currentBuildRelease
			set.addRange(start, end)
		} else if strings.Contains(piece, "-") {
			limits := strings.SplitN(piece, "-", 2)
			start, err := nameToRelease(limits[0])
			if err != nil {
				return nil, err
			}

			end, err := nameToRelease(limits[1])
			if err != nil {
				return nil, err
			}

			if start.ordinal > end.ordinal {
				return nil, fmt.Errorf("invalid closed range, start release %q is later than end release %q", start.name, end.name)
			}

			set.addRange(start, end)
		} else {
			release, err := nameToRelease(piece)
			if err != nil {
				return nil, err
			}
			set.addItem(release)
		}
	}

	return set, nil
}
