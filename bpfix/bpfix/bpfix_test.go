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

// This file implements the logic of bpfix and also provides a programmatic interface

package bpfix

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/blueprint/parser"
)

// TODO(jeffrygaston) remove this when position is removed from ParseNode (in b/38325146) and we can directly do reflect.DeepEqual
func printListOfStrings(items []string) (text string) {
	if items == nil {
		return "nil"
	}
	if len(items) == 0 {
		return "[]"
	}
	return fmt.Sprintf("[\"%s\"]", strings.Join(items, "\", \""))

}

func parseForTest(t *testing.T, input string) (tree *parser.File) {
	tree, errs := parser.Parse("", strings.NewReader(input), parser.NewScope(nil))
	if len(errs) > 0 {
		t.Fatalf(fmt.Sprintf("%v errors parsing \n%v\n: %v", len(errs), input, errs))
	}
	return tree
}

func buildInclusionsTree(t *testing.T, local_include_dirs []string, export_include_dirs []string) (file *parser.File) {
	// TODO(jeffrygaston) use the builder class when b/38325146 is done
	input := fmt.Sprintf(`cc_library_shared {
	    name: "iAmAModule",
	    local_include_dirs: %s,
	    export_include_dirs: %s,
	}
	`,
		printListOfStrings(local_include_dirs), printListOfStrings(export_include_dirs))
	return parseForTest(t, input)
}

func getProp(t *testing.T, tree *parser.File, propertyName string) (propertyValue *parser.Property) {
	// lookup legacy property
	mod := tree.Defs[0].(*parser.Module)
	propertyValue, found := mod.GetProperty(propertyName)
	if !found {
		t.Errorf("property " + propertyName + " not found in parse tree")
		t.FailNow()
	}

	return propertyValue
}

func toListOfStrings(list *parser.List) []string {
	if list == nil {
		return nil
	}
	values := list.Values
	if values == nil {
		return nil
	}
	results := make([]string, 0, len(values))
	for _, expr := range values {
		str := expr.(*parser.String)
		results = append(results, str.Value)
	}
	return results
}

func implOfFilterListTest(t *testing.T, local_include_dirs []string, export_include_dirs []string, expectedResult []string) {
	// build tree
	tree := buildInclusionsTree(t, local_include_dirs, export_include_dirs)

	// apply simplifications
	tree, err := simplifyKnownPropertiesDuplicatingEachOther(tree)
	if err != nil {
		t.Fatal(err)
	}

	// lookup legacy property
	result := getProp(t, tree, "local_include_dirs")

	// check that the value for the legacy property was updated to the correct value
	errorHeader := fmt.Sprintf("\nFailed to correctly simplify key 'local_include_dirs' in the presence of 'export_include_dirs.'\n"+
		"original local_include_dirs: %#v\n"+
		"original export_include_dirs: %#v\n"+
		"expected result: %#v\n"+
		"actual result: ",
		local_include_dirs, export_include_dirs, expectedResult)

	listResult, ok := result.Value.(*parser.List)

	if listResult != nil && !ok {
		t.Fatalf("%sproperty is not a list: %v", errorHeader, listResult)
	}

	actualValues := toListOfStrings(listResult)

	if !reflect.DeepEqual(actualValues, expectedResult) {
		t.Fatalf("%s%#v\nlists are different", errorHeader, actualValues)
	}
}

func TestSimplifyKnownVariablesDuplicatingEachOther(t *testing.T) {
	// TODO use []Expression{} once buildInclusionsTree above can support it (which is after b/38325146 is done)
	implOfFilterListTest(t, []string{"include"}, []string{"include"}, []string{})
	implOfFilterListTest(t, []string{"include1"}, []string{"include2"}, []string{"include1"})
	implOfFilterListTest(t, []string{"include1", "include2", "include3", "include4"}, []string{"include2"},
		[]string{"include1", "include3", "include4"})
	implOfFilterListTest(t, nil, []string{"include"}, nil)
	implOfFilterListTest(t, nil, nil, nil)
}

func TestKeepEmptyListPointer(t *testing.T) {
	// make tree
	input := fmt.Sprintf(`cc_defaults {
	    name: "iAmSomeDefaults",
	    system_shared_libs: [],
	}
	`)
	tree := parseForTest(t, input)

	// do simplification
	fixed, err := removePropertiesHavingTheirDefaultValues(tree)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// check results
	prop := getProp(t, fixed, "system_shared_libs")
	if prop == nil {
		t.Fatalf("incorrectly modified list pointer; was [], became removed")
	}
}
