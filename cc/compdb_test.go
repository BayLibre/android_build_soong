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

package cc

import (
	"testing"

	"github.com/google/blueprint/proptools"
)

func testEscape(s string) string {
	return proptools.ShellEscape(s)
}
func unescape(s string) string {
	var arg []string
	arg = append(arg, s)
	return unquote(arg)[0]
}
func TestUnquote(t *testing.T) {
	vals := []string{
		"abc",
		"'abc'",
		`'`,
		`\'`,
		`'\'`,
		`a b c`,
		`a c`,
	}
	for _, val := range vals {
		if val != unescape(testEscape(val)) {
			t.Errorf("failed to unquote %s -> %s -> %s", val, testEscape(val), unescape(testEscape(val)))
		}
	}
}

type testcase struct {
	orig string
	res  []string
}

func equal(a, b []string, t *testing.T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i] != b[i] {
			t.Errorf("Index %d diff %s != %s", i, a[i], b[i])
			return false
		}
	}
	return true
}

func TestShellSplit(t *testing.T) {
	vals := []testcase{
		testcase{"   a b   c d   ", []string{"a", "b", "c", "d"}},
		testcase{"   'a b'  c d   ", []string{"'a b'", "c", "d"}},
		testcase{"'a   b  c'  c 'd   '", []string{"'a   b  c'", "c", "'d   '"}},
	}
	for _, val := range vals {
		if !equal(val.res, shellSplit(val.orig), t) {
			t.Errorf("Failed to split %s -> %s != %s", val.orig, shellSplit(val.orig), val.res)
		}
	}
}
