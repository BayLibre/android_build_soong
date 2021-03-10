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
package android

import "testing"

func TestConvertAllModulesInPackage(t *testing.T) {
	testCases := []struct {
		prefixes   BazelConversionConfig
		packageDir string
	}{
		{
			prefixes: BazelConversionConfig{
				"a": ConvertSubtree,
			},
			packageDir: "a",
		},
		{
			prefixes: BazelConversionConfig{
				"a/b": ConvertSubtree,
			},
			packageDir: "a/b",
		},
		{
			prefixes: BazelConversionConfig{
				"a/b":   ConvertSubtree,
				"a/b/c": ConvertSubtree,
			},
			packageDir: "a/b",
		},
		{
			prefixes: BazelConversionConfig{
				"a":     ConvertSubtree,
				"d/e/f": ConvertSubtree,
			},
			packageDir: "a/b",
		},
		{
			prefixes: BazelConversionConfig{
				"a":     ConvertModuleOptIn,
				"a/b":   ConvertSubtree,
				"a/b/c": ConvertModuleOptIn,
			},
			packageDir: "a/b",
		},
		{
			prefixes: BazelConversionConfig{
				"a":     ConvertSubtree,
				"a/b":   ConvertModuleOptIn,
				"a/b/c": ConvertSubtree,
			},
			packageDir: "a",
		},
	}

	for _, test := range testCases {
		if !convertAllModulesInPackage(test.packageDir, test.prefixes) {
			t.Errorf("Expected to convert all modules in %s based on %v, but failed.", test.packageDir, test.prefixes)
		}
	}
}

func TestModuleOptIn(t *testing.T) {
	testCases := []struct {
		prefixes   BazelConversionConfig
		packageDir string
	}{
		{
			prefixes: BazelConversionConfig{
				"a/b": ConvertModuleOptIn,
			},
			packageDir: "a/b",
		},
		{
			prefixes: BazelConversionConfig{
				"a":   ConvertModuleOptIn,
				"a/b": ConvertSubtree,
			},
			packageDir: "a",
		},
		{
			prefixes: BazelConversionConfig{
				"a/b": ConvertSubtree,
			},
			packageDir: "a", // opt-in by default
		},
		{
			prefixes: BazelConversionConfig{
				"a/b/c": ConvertSubtree,
			},
			packageDir: "a/b",
		},
		{
			prefixes: BazelConversionConfig{
				"a":     ConvertSubtree,
				"d/e/f": ConvertSubtree,
			},
			packageDir: "foo/bar",
		},
		{
			prefixes: BazelConversionConfig{
				"a":     ConvertSubtree,
				"a/b":   ConvertModuleOptIn,
				"a/b/c": ConvertSubtree,
			},
			packageDir: "a/b",
		},
		{
			prefixes: BazelConversionConfig{
				"a":     ConvertModuleOptIn,
				"a/b":   ConvertSubtree,
				"a/b/c": ConvertModuleOptIn,
			},
			packageDir: "a",
		},
	}

	for _, test := range testCases {
		if convertAllModulesInPackage(test.packageDir, test.prefixes) {
			t.Errorf("Expected to allow module opt-in in %s based on %v, but failed.", test.packageDir, test.prefixes)
		}
	}
}
