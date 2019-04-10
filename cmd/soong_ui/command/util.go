// Copyright 2019 Google Inc. All rights reserved.
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

package command

// indexList returns the index of first found s. -1 is return if s is not
// found.
func indexList(s string, list []string) int {
	for i, l := range list {
		if l == s {
			return i
		}
	}
	return -1
}

// inList returns true if one or more of s is in the list.
func inList(s string, list []string) bool {
	return indexList(s, list) != -1
}

// removeFromList removes all instances of string s from the list.
func removeFromList(s string, list []string) []string {
	c := 0
	for i, l := range list {
		if l != s {
			list[c] = list[i]
			c++
		}
	}
	if c == 0 {
		return []string{}
	}
	return list[0:c]
}
