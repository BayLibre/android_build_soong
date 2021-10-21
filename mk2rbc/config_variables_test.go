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

package mk2rbc

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

type testVar struct {
	name string
	cl   VarClass
	ty   StarlarkType
}

type testVariables struct {
	v []testVar
}

func (v *testVariables) NewVariable(name string, varClass VarClass, valueType StarlarkType) {
	v.v = append(v.v, testVar{name, varClass, valueType})
}

// getTestDirectory returns the test directory, which should be the test/ subdirectory
func getTestDirectory() string {
	_, myFile, _, _ := runtime.Caller(1)
	return filepath.Join(filepath.Dir(myFile), "test")
}

func TestConfigVariables(t *testing.T) {
	testFile := filepath.Join(getTestDirectory(), "config_variables.mk.test")
	var actual testVariables
	if err := FindConfigVariables(testFile, &actual); err != nil {
		t.Fatal(err)
	}
	expected := testVariables{[]testVar{
		{"PRODUCT_NAME", VarClassConfig, StarlarkTypeUnknown},
		{"PRODUCT_MODEL", VarClassConfig, StarlarkTypeUnknown},
		{"PRODUCT_LOCALES", VarClassConfig, StarlarkTypeList},
		{"PRODUCT_AAPT_CONFIG", VarClassConfig, StarlarkTypeList},
		{"PRODUCT_AAPT_PREF_CONFIG", VarClassConfig, StarlarkTypeUnknown},
	}}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("\nExpected: %v\n  Actual: %v", expected, actual)
	}
}
