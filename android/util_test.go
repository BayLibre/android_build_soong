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

package android

import (
	"reflect"
	"testing"
)

var firstUniqueStringsTestCases = []struct {
	in  []string
	out []string
}{
	{
		in:  []string{"a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b", "a"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"b", "a", "a"},
		out: []string{"b", "a"},
	},
	{
		in:  []string{"a", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "b", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"liblog", "libdl", "libc++", "libdl", "libc", "libm"},
		out: []string{"liblog", "libdl", "libc++", "libc", "libm"},
	},
}

func TestFirstUniqueStrings(t *testing.T) {
	for _, testCase := range firstUniqueStringsTestCases {
		out := FirstUniqueStrings(testCase.in)
		if !reflect.DeepEqual(out, testCase.out) {
			t.Errorf("incorrect output:")
			t.Errorf("     input: %#v", testCase.in)
			t.Errorf("  expected: %#v", testCase.out)
			t.Errorf("       got: %#v", out)
		}
	}
}

var lastUniqueStringsTestCases = []struct {
	in  []string
	out []string
}{
	{
		in:  []string{"a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b", "a"},
		out: []string{"b", "a"},
	},
	{
		in:  []string{"b", "a", "a"},
		out: []string{"b", "a"},
	},
	{
		in:  []string{"a", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "b", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"liblog", "libdl", "libc++", "libdl", "libc", "libm"},
		out: []string{"liblog", "libc++", "libdl", "libc", "libm"},
	},
}

func TestLastUniqueStrings(t *testing.T) {
	for _, testCase := range lastUniqueStringsTestCases {
		out := LastUniqueStrings(testCase.in)
		if !reflect.DeepEqual(out, testCase.out) {
			t.Errorf("incorrect output:")
			t.Errorf("     input: %#v", testCase.in)
			t.Errorf("  expected: %#v", testCase.out)
			t.Errorf("       got: %#v", out)
		}
	}
}

func TestJoinWithPrefix(t *testing.T) {
	expected := ""
	out := JoinWithPrefix([]string{}, "prefix:")
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}

	expected = "prefix:a"
	out = JoinWithPrefix([]string{"a"}, "prefix:")
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}

	expected = "prefix:a prefix:b"
	out = JoinWithPrefix([]string{"a", "b"}, "prefix:")
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}
}

func TestIndexList(t *testing.T) {
	input := []string{"a", "b", "c"}
	for expected, key := range input {
		out := IndexList(key, input)
		if out != expected {
			t.Errorf("incorrect output:")
			t.Errorf("  expected: %#v", expected)
			t.Errorf("       got: %#v", out)
		}
	}

	expected := -1
	out := IndexList("does_not_exist", input)
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}
}

func TestInList(t *testing.T) {
	input := []string{"a"}

	expected := true
	out := InList("a", input)
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}

	expected = false
	out = InList("does_not_exist", input)
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}
}

func TestPrefixInList(t *testing.T) {
	input := []string{"a", "b"}

	expected := true
	out := PrefixInList("a-example", input)
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}

	expected = true
	out = PrefixInList("b-example", input)
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}

	expected = false
	out = PrefixInList("c-example", input)
	if out != expected {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}
}

func TestFilterList(t *testing.T) {
	input := []string{"a", "b", "c", "c", "b", "d", "a"}
	filter := []string{"a", "c"}
	remainder, filtered := FilterList(input, filter)

	expected := []string{"b", "b", "d"}
	if !reflect.DeepEqual(remainder, expected) {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", remainder)
	}

	expected = []string{"a", "c", "c", "a"}
	if !reflect.DeepEqual(filtered, expected) {
		t.Errorf("incorrect output:")
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", filtered)
	}
}

func TestRemoveListFromList(t *testing.T) {
	input := []string{"a", "b", "c", "d", "a", "c", "d"}
	expected := []string{"b", "d", "d"}
	out := RemoveListFromList(input, []string{"a", "c"})
	if !reflect.DeepEqual(out, expected) {
		t.Errorf("incorrect output:")
		t.Errorf("     input: %#v", input)
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}
}
