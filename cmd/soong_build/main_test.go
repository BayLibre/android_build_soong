// Copyright 2021 Google Inc. All rights reserved.
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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"android/soong/android"
	"github.com/google/blueprint"
)

var buildDir string

func setUp() {
	var err error
	buildDir, err = ioutil.TempDir("", "soong_android_test")
	if err != nil {
		panic(err)
	}
}

func tearDown() {
	os.RemoveAll(buildDir)
}

func TestMain(m *testing.M) {
	run := func() int {
		setUp()
		defer tearDown()

		return m.Run()
	}

	os.Exit(run())
}

// Make sure that the order of registration for those build components where order matters matches
// between the runtime and test environments.
//
// The build components where order matters are:
// * mutators
// * pre-singleton
// * singletons
//
// Their order matters because they are invoked in the order in which they are registered and the
// output from one is the input to the next.
func TestRegistrationOrder(t *testing.T) {
	result := android.NewFixtureFactory(&buildDir,
		// Use all the fixture preparers that have been registered in all the packages.
		android.AllRegisteredFixturePreparers()...,
	).RunTest(t)

	registeredLists := []struct {
		blueprintContextField string
		componentType         string
	}{
		// Doesn't check earlyMutatorInfo as they are deprecated and not used.
		{
			blueprintContextField: "mutatorInfo",
			componentType:         "mutator",
		},
		{
			blueprintContextField: "preSingletonInfo",
			componentType:         "pre-singleton",
		},
		{
			blueprintContextField: "singletonInfo",
			componentType:         "singleton",
		},
	}

	for _, r := range registeredLists {
		fieldName := r.blueprintContextField
		componentType := r.componentType

		t.Run("checking order of "+componentType, func(t *testing.T) {
			// The component names registered in the test context.
			testComponentNames := registeredComponentNames(result.TestContext.Context.Context, fieldName)

			context := newContext(android.TestConfig(buildDir, nil, "", nil))
			runtimeComponentNames := registeredComponentNames(context.Context, fieldName)

			lastMatching := -1
			matchCount := 0
			for i, j := 0, 0; i < len(testComponentNames) && j < len(runtimeComponentNames); {
				test := testComponentNames[i]
				runtime := runtimeComponentNames[j]

				if test == runtime {
					testComponentNames[i] = test + fmt.Sprintf(" <-- matched with runtime %s %d", componentType, j)
					runtimeComponentNames[j] = runtime + fmt.Sprintf(" <-- matched with test %s %d", componentType, i)
					lastMatching = i
					i += 1
					j += 1
					matchCount += 1
				} else {
					// Assume that the test list is in the same order as the runtime list but the runtime list
					// contains some components that are not present in the tests. So, skip the runtime component to
					// try and find the next one that matches the current test component.
					j += 1
				}
			}

			if matchCount != len(testComponentNames) {
				// The test component names were not all matched with a runtime component name so there must either
				// be a component present in the test that is not present in the runtime or they must be in the
				// wrong order.
				testComponentNames[lastMatching+1] = testComponentNames[lastMatching+1] + " <--- unmatched"
				t.Errorf("test %[1]ss:\n    %[2]s\nruntime %[1]ss\n    %[3]s\n", componentType, strings.Join(testComponentNames, "\n    "), strings.Join(runtimeComponentNames, "\n    "))
			}
		})
	}
}

// registeredComponentNames uses reflection to retrieve the names of the registered components
// directly from the blueprint.Context object as there is no accessor.
func registeredComponentNames(context *blueprint.Context, fieldName string) []string {
	contextStruct := reflect.ValueOf(context).Elem()
	componentInfoArray := contextStruct.FieldByName(fieldName)
	componentCount := componentInfoArray.Len()
	componentNames := []string{}
	for m := 0; m < componentCount; m += 1 {
		componentName := componentInfoArray.Index(m).Elem().FieldByName("name").String()
		componentNames = append(componentNames, componentName)
	}
	return componentNames
}
