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

package java

import (
	"strings"
	"testing"
)

func TestHasAnySubstring(t *testing.T) {
	testcases := []struct {
		name       string
		str        string
		substrings []string
		want       bool
	}{
		{
			name:       "empty string & no substrings",
			str:        "",
			substrings: []string{},
			want:       false,
		},
		{
			name:       "empty string & empty substring",
			str:        "",
			substrings: []string{""},
			want:       true,
		},
		{
			name:       "no substrings",
			str:        "input",
			substrings: []string{},
			want:       false,
		},
		{
			name:       "empty substring",
			str:        "input",
			substrings: []string{"", "xyz"},
			want:       true,
		},
		{
			name:       "no valid substrings",
			str:        "input",
			substrings: []string{"x", "y", "z"},
			want:       false,
		},
		{
			name:       "all valid substrings",
			str:        "input",
			substrings: []string{"i", "n", "p"},
			want:       true,
		},
		{
			name:       "one valid substring",
			str:        "input",
			substrings: []string{"x", "y", "z", "i"},
			want:       true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasAnySubstring(tc.str, tc.substrings...); got != tc.want {
				t.Errorf("hasAnySubstring() = %t, want = %t", got, tc.want)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	alwaysTrue := func(string) bool { return true }
	alwaysFalse := func(string) bool { return false }
	containsA := func(input string) bool { return strings.Contains(input, "a") }

	testcases := []struct {
		name       string
		list       []string
		predicates []func(string) bool
		want       []string
	}{
		{
			name:       "no predicates",
			list:       []string{"foo", "bar", "baz"},
			predicates: []func(string) bool{},
			want:       []string{"foo", "bar", "baz"},
		},
		{
			name:       "always true predicate",
			list:       []string{"foo", "bar", "baz"},
			predicates: []func(string) bool{alwaysTrue},
			want:       []string{"foo", "bar", "baz"},
		},
		{
			name:       "always false predicate",
			list:       []string{"foo", "bar", "baz"},
			predicates: []func(string) bool{alwaysFalse},
			want:       []string{},
		},
		{
			name:       "contains 'a' predicate",
			list:       []string{"foo", "bar", "baz"},
			predicates: []func(string) bool{containsA},
			want:       []string{"bar", "baz"},
		},
		{
			name:       "always true & contains 'a'",
			list:       []string{"foo", "bar", "baz"},
			predicates: []func(string) bool{alwaysTrue, containsA},
			want:       []string{"bar", "baz"},
		},
	}

	stringSlicesEqual := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filter(tc.list, tc.predicates...); !stringSlicesEqual(got, tc.want) {
				t.Errorf("filter() = %v, want = %v", got, tc.want)
			}
		})
	}
}
