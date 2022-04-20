// Copyright 2022 Google Inc. All rights reserved.
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
	"fmt"
	"testing"

	"android/soong/android"
)

func TestWithTidy(t *testing.T) {
	// When WITH_TIDY=1 or (ALLOW_LOCAL_TIDY_TRUE=1 and local tidy:true)
	// a C++ library should depend on .tidy files.
	testCases := []struct {
		withTidy, allowLocalTidyTrue string // "_" means undefined
		needTidyFile                 []bool // for {libfoo_0, libfoo_1} and {libbar_0, libbar_1}
	}{
		{"_", "_", []bool{false, false}},
		{"_", "0", []bool{false, false}},
		{"_", "1", []bool{false, true}},
		{"_", "true", []bool{false, true}},
		{"0", "_", []bool{false, false}},
		{"0", "1", []bool{false, true}},
		{"1", "_", []bool{true, true}},
		{"1", "false", []bool{true, true}},
		{"1", "1", []bool{true, true}},
		{"true", "_", []bool{true, true}},
	}
	bp := `
		cc_library_shared {
			name: "libfoo_0", // depends on .tidy if WITH_TIDY=1
			srcs: ["foo.c"],
		}
		cc_library_shared { // depends on .tidy if WITH_TIDY=1 or ALLOW_LOCAL_TIDY_TRUE=1
			name: "libfoo_1",
			srcs: ["foo.c"],
			tidy: true,
		}
		cc_library_static {
			name: "libbar_0", // depends on .tidy if WITH_TIDY=1
			srcs: ["bar.c"],
		}
		cc_library_static { // depends on .tidy if WITH_TIDY=1 or ALLOW_LOCAL_TIDY_TRUE=1
			name: "libbar_1",
			srcs: ["bar.c"],
			tidy: true,
		}`
	for index, test := range testCases {
		testName := fmt.Sprintf("case%d,%v,%v", index, test.withTidy, test.allowLocalTidyTrue)
		t.Run(testName, func(t *testing.T) {
			testEnv := map[string]string{}
			if test.withTidy != "_" {
				testEnv["WITH_TIDY"] = test.withTidy
			}
			if test.allowLocalTidyTrue != "_" {
				testEnv["ALLOW_LOCAL_TIDY_TRUE"] = test.allowLocalTidyTrue
			}
			ctx := android.GroupFixturePreparers(prepareForCcTest, android.FixtureMergeEnv(testEnv)).RunTestWithBp(t, bp)
			for n := 0; n < 2; n++ {
				checkLibraryRule := func(foo, variant, ruleName string) {
					libName := fmt.Sprintf("lib%s_%d", foo, n)
					tidyFile := "out/soong/.intermediates/" + libName + "/" + variant + "/obj/" + foo + ".tidy"
					depFiles := ctx.ModuleForTests(libName, variant).Rule(ruleName).Validations.Strings()
					if test.needTidyFile[n] {
						android.AssertStringListContains(t, libName+" needs .tidy file", depFiles, tidyFile)
					} else {
						android.AssertStringListDoesNotContain(t, libName+" does not need .tidy file", depFiles, tidyFile)
					}
				}
				checkLibraryRule("foo", "android_arm64_armv8-a_shared", "ld")
				checkLibraryRule("bar", "android_arm64_armv8-a_static", "ar")
			}
		})
	}
}
