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
		prefixes   Bp2BuildPackageConversionConfig
		packageDir string
	}{
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a": AllModules,
			},
			packageDir: "a",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a/b": AllModules,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a/b":   AllModules,
				"a/b/c": AllModules,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a":     AllModules,
				"d/e/f": AllModules,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a":     ModuleOptIn,
				"a/b":   AllModules,
				"a/b/c": ModuleOptIn,
			},
			packageDir: "a/b",
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
		prefixes   Bp2BuildPackageConversionConfig
		packageDir string
	}{
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a/b": ModuleOptIn,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a": ModuleOptIn,
			},
			packageDir: "a",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a/b": AllModules,
			},
			packageDir: "a",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a/b/c": AllModules,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a":     AllModules,
				"d/e/f": AllModules,
			},
			packageDir: "foo/bar",
		},
		{
			prefixes: Bp2BuildPackageConversionConfig{
				"a":     AllModules,
				"a/b":   ModuleOptIn,
				"a/b/c": AllModules,
			},
			packageDir: "a/b",
		},
	}

	for _, test := range testCases {
		if convertAllModulesInPackage(test.packageDir, test.prefixes) {
			t.Errorf("Expected to allow module opt-in in %s based on %v, but failed.", test.packageDir, test.prefixes)
		}
	}
}
