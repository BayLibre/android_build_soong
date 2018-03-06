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
	check := func(input []string, prefix string, expected string) {
		out := JoinWithPrefix(input, prefix)
		if out != expected {
			t.Errorf("incorrect output:")
			t.Errorf("     input: %#v", input)
			t.Errorf("    prefix: %#v", prefix)
			t.Errorf("  expected: %#v", expected)
			t.Errorf("       got: %#v", out)
		}
	}

	check([]string{}, "prefix:", "")
	check([]string{"a"}, "prefix:", "prefix:a")
	check([]string{"a", "b"}, "prefix:", "prefix:a prefix:b")
}

func TestIndexList(t *testing.T) {
	check := func(key string, input []string, expected int) {
		out := IndexList(key, input)
		if out != expected {
			t.Errorf("incorrect output:")
			t.Errorf("       key: %#v", key)
			t.Errorf("     input: %#v", input)
			t.Errorf("  expected: %#v", expected)
			t.Errorf("       got: %#v", out)
		}
	}

	input := []string{"a", "b", "c"}
	for expected, key := range input {
		check(key, input, expected)
	}
	check("X", input, -1)
}

func TestInList(t *testing.T) {
	check := func(key string, input []string, expected bool) {
		out := InList(key, input)
		if out != expected {
			t.Errorf("incorrect output:")
			t.Errorf("       key: %#v", key)
			t.Errorf("     input: %#v", input)
			t.Errorf("  expected: %#v", expected)
			t.Errorf("       got: %#v", out)
		}
	}

	input := []string{"a"}
	check("a", input, true)
	check("X", input, false)
}

func TestPrefixInList(t *testing.T) {
	check := func(str string, prefixes []string, expected bool) {
		out := PrefixInList(str, prefixes)
		if out != expected {
			t.Errorf("incorrect output:")
			t.Errorf("       str: %#v", str)
			t.Errorf("  prefixes: %#v", prefixes)
			t.Errorf("  expected: %#v", expected)
			t.Errorf("       got: %#v", out)
		}
	}

	prefixes := []string{"a", "b"}
	check("a-example", prefixes, true)
	check("b-example", prefixes, true)
	check("c-example", prefixes, false)
}

func TestFilterList(t *testing.T) {
	input := []string{"a", "b", "c", "c", "b", "d", "a"}
	filter := []string{"a", "c"}
	remainder, filtered := FilterList(input, filter)

	expected := []string{"b", "b", "d"}
	if !reflect.DeepEqual(remainder, expected) {
		t.Errorf("incorrect remainder output:")
		t.Errorf("     input: %#v", input)
		t.Errorf("    filter: %#v", filter)
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", remainder)
	}

	expected = []string{"a", "c", "c", "a"}
	if !reflect.DeepEqual(filtered, expected) {
		t.Errorf("incorrect filtered output:")
		t.Errorf("     input: %#v", input)
		t.Errorf("    filter: %#v", filter)
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", filtered)
	}
}

func TestRemoveListFromList(t *testing.T) {
	input := []string{"a", "b", "c", "d", "a", "c", "d"}
	filter := []string{"a", "c"}
	expected := []string{"b", "d", "d"}
	out := RemoveListFromList(input, filter)
	if !reflect.DeepEqual(out, expected) {
		t.Errorf("incorrect output:")
		t.Errorf("     input: %#v", input)
		t.Errorf("    filter: %#v", filter)
		t.Errorf("  expected: %#v", expected)
		t.Errorf("       got: %#v", out)
	}
}

func TestRemoveFromList(t *testing.T) {
	check := func(key string, input []string, expectedN int, expectedOut []string) {
		n, out := RemoveFromList(key, input)
		if n != expectedN {
			t.Errorf("incorrect output:")
			t.Errorf("       key: %#v", key)
			t.Errorf("     input: %#v", input)
			t.Errorf("  expected: %#v", expectedN)
			t.Errorf("       got: %#v", n)
		}
		if !reflect.DeepEqual(out, expectedOut) {
			t.Errorf("incorrect output:")
			t.Errorf("       key: %#v", key)
			t.Errorf("     input: %#v", input)
			t.Errorf("  expected: %#v", expectedOut)
			t.Errorf("       got: %#v", out)
		}
	}

	input := []string{"a", "b", "a", "c", "a"}
	check("a", input, 3, []string{"b", "c"})
	check("d", input, 0, input)
}
