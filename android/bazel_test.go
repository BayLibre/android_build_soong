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

import (
	"android/soong/bazel"
	"fmt"
	"testing"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func TestConvertAllModulesInPackage(t *testing.T) {
	testCases := []struct {
		prefixes   Bp2BuildConfig
		packageDir string
	}{
		{
			prefixes: Bp2BuildConfig{
				"a": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a",
		},
		{
			prefixes: Bp2BuildConfig{
				"a/b": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildConfig{
				"a/b":   Bp2BuildDefaultTrueRecursively,
				"a/b/c": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildConfig{
				"a":     Bp2BuildDefaultTrueRecursively,
				"d/e/f": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildConfig{
				"a":     Bp2BuildDefaultFalse,
				"a/b":   Bp2BuildDefaultTrueRecursively,
				"a/b/c": Bp2BuildDefaultFalse,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildConfig{
				"a":     Bp2BuildDefaultTrueRecursively,
				"a/b":   Bp2BuildDefaultFalse,
				"a/b/c": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a",
		},
	}

	for _, test := range testCases {
		if ok, _ := bp2buildDefaultTrueRecursively(test.packageDir, test.prefixes); !ok {
			t.Errorf("Expected to convert all modules in %s based on %v, but failed.", test.packageDir, test.prefixes)
		}
	}
}

func TestModuleOptIn(t *testing.T) {
	testCases := []struct {
		prefixes   Bp2BuildConfig
		packageDir string
	}{
		{
			prefixes: Bp2BuildConfig{
				"a/b": Bp2BuildDefaultFalse,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildConfig{
				"a":   Bp2BuildDefaultFalse,
				"a/b": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a",
		},
		{
			prefixes: Bp2BuildConfig{
				"a/b": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a", // opt-in by default
		},
		{
			prefixes: Bp2BuildConfig{
				"a/b/c": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildConfig{
				"a":     Bp2BuildDefaultTrueRecursively,
				"d/e/f": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "foo/bar",
		},
		{
			prefixes: Bp2BuildConfig{
				"a":     Bp2BuildDefaultTrueRecursively,
				"a/b":   Bp2BuildDefaultFalse,
				"a/b/c": Bp2BuildDefaultTrueRecursively,
			},
			packageDir: "a/b",
		},
		{
			prefixes: Bp2BuildConfig{
				"a":     Bp2BuildDefaultFalse,
				"a/b":   Bp2BuildDefaultTrueRecursively,
				"a/b/c": Bp2BuildDefaultFalse,
			},
			packageDir: "a",
		},
	}

	for _, test := range testCases {
		if ok, _ := bp2buildDefaultTrueRecursively(test.packageDir, test.prefixes); ok {
			t.Errorf("Expected to allow module opt-in in %s based on %v, but failed.", test.packageDir, test.prefixes)
		}
	}
}

type TestBazelModule struct {
	bazel.TestModuleInfo
	BazelModuleBase
}

var _ blueprint.Module = TestBazelModule{}

func (m TestBazelModule) Name() string {
	return m.Name_
}
func (m TestBazelModule) GenerateBuildActions(blueprint.ModuleContext) {
}

type TestBazelConversionContext struct {
	omc    bazel.OtherModuleTestContext
	errors []string
}

var _ BazelOtherModuleContext = &TestBazelConversionContext{}

func (bcc *TestBazelConversionContext) OtherModuleType(m blueprint.Module) string {
	return bcc.omc.OtherModuleType(m)
}
func (bcc *TestBazelConversionContext) OtherModuleName(m blueprint.Module) string {
	return bcc.omc.OtherModuleName(m)
}
func (bcc *TestBazelConversionContext) OtherModuleDir(m blueprint.Module) string {
	return bcc.omc.OtherModuleDir(m)
}
func (bcc *TestBazelConversionContext) ModuleErrorf(format string, args ...interface{}) {
	bcc.errors = append(bcc.errors, fmt.Sprintf(format, args...))
}

var bazelableBazelModuleBase = BazelModuleBase{
	bazelProperties: properties{
		Bazel_module: bazelModuleProperties{
			CanConvertToBazel:  true,
			Bp2build_available: proptools.BoolPtr(true),
		},
	},
}

func TestBp2BuildAllowlist(t *testing.T) {
	testCases := []struct {
		description    string
		shouldConvert  bool
		expectedErrors []string
		module         TestBazelModule
		allowlist      Bp2BuildConversionAllowlist
	}{
		{
			description:   "allowlist enables module",
			shouldConvert: true,
			module: TestBazelModule{
				TestModuleInfo: bazel.TestModuleInfo{
					Name_: "foo",
					Typ:   "rule1",
					Dir:   "dir1",
				},
				BazelModuleBase: bazelableBazelModuleBase,
			},
			allowlist: Bp2BuildConversionAllowlist{
				bp2buildModuleAlwaysConvert: map[string]bool{
					"foo": true,
				},
			},
		},
		{
			description:    "module in name allowlist and type allowlist fails",
			shouldConvert:  false,
			expectedErrors: []string{"A module cannot be in bp2buildModuleAlwaysConvert and also be in bp2buildModuleTypeAlwaysConvert"},
			module: TestBazelModule{
				TestModuleInfo: bazel.TestModuleInfo{
					Name_: "foo",
					Typ:   "rule1",
					Dir:   "dir1",
				},
				BazelModuleBase: bazelableBazelModuleBase,
			},
			allowlist: Bp2BuildConversionAllowlist{
				bp2buildModuleAlwaysConvert: map[string]bool{
					"foo": true,
				},
				bp2buildModuleTypeAlwaysConvert: map[string]bool{
					"rule1": true,
				},
			},
		},
		{
			description:    "module in allowlist and denylist fails",
			shouldConvert:  false,
			expectedErrors: []string{"a module cannot be in bp2buildModuleDoNotConvert and also be in bp2buildModuleAlwaysConvert"},
			module: TestBazelModule{
				TestModuleInfo: bazel.TestModuleInfo{
					Name_: "foo",
					Typ:   "rule1",
					Dir:   "dir1",
				},
				BazelModuleBase: bazelableBazelModuleBase,
			},
			allowlist: Bp2BuildConversionAllowlist{
				bp2buildModuleAlwaysConvert: map[string]bool{
					"foo": true,
				},
				bp2buildModuleDoNotConvert: map[string]bool{
					"foo": true,
				},
			},
		},
		{
			description:    "module in allowlist and existing BUILD file",
			shouldConvert:  false,
			expectedErrors: []string{"A module cannot be in a directory listed in bp2buildKeepExistingBuildFile and also be in bp2buildModuleAlwaysConvert. Directory: 'existing/build/dir'"},
			module: TestBazelModule{
				TestModuleInfo: bazel.TestModuleInfo{
					Name_: "foo",
					Typ:   "rule1",
					Dir:   "existing/build/dir",
				},
				BazelModuleBase: bazelableBazelModuleBase,
			},
			allowlist: Bp2BuildConversionAllowlist{
				bp2buildModuleAlwaysConvert: map[string]bool{
					"foo": true,
				},
				bp2buildKeepExistingBuildFile: map[string]bool{
					"existing/build/dir": true,
				},
			},
		},
		{
			description:    "module allowlist and enabled directory",
			shouldConvert:  false,
			expectedErrors: []string{"A module cannot be in a directory marked Bp2BuildDefaultTrue or Bp2BuildDefaultTrueRecursively and also be in bp2buildModuleAlwaysConvert. Directory: 'existing/build/dir'"},
			module: TestBazelModule{
				TestModuleInfo: bazel.TestModuleInfo{
					Name_: "foo",
					Typ:   "rule1",
					Dir:   "existing/build/dir",
				},
				BazelModuleBase: bazelableBazelModuleBase,
			},
			allowlist: Bp2BuildConversionAllowlist{
				bp2buildModuleAlwaysConvert: map[string]bool{
					"foo": true,
				},
				bp2buildDefaultConfig: allowlists.Bp2BuildConfig{
					"existing/build/dir": allowlists.Bp2BuildDefaultTrue,
				},
			},
		},
		{
			description:    "module allowlist and enabled subdirectory",
			shouldConvert:  false,
			expectedErrors: []string{"A module cannot be in a directory marked Bp2BuildDefaultTrue or Bp2BuildDefaultTrueRecursively and also be in bp2buildModuleAlwaysConvert. Directory: 'existing/build/dir'"},
			module: TestBazelModule{
				TestModuleInfo: bazel.TestModuleInfo{
					Name_: "foo",
					Typ:   "rule1",
					Dir:   "existing/build/dir/subdir",
				},
				BazelModuleBase: bazelableBazelModuleBase,
			},
			allowlist: Bp2BuildConversionAllowlist{
				bp2buildModuleAlwaysConvert: map[string]bool{
					"foo": true,
				},
				bp2buildDefaultConfig: allowlists.Bp2BuildConfig{
					"existing/build/dir": allowlists.Bp2BuildDefaultTrueRecursively,
				},
			},
		},
		{
			description:   "module enabled in unit test can always be converted",
			shouldConvert: true,
			module: TestBazelModule{
				TestModuleInfo: bazel.TestModuleInfo{
					Name_: "foo",
					Typ:   "rule1",
					Dir:   ".",
				},
				BazelModuleBase: bazelableBazelModuleBase,
			},
			allowlist: Bp2BuildConversionAllowlist{
				bp2buildModuleAlwaysConvert: map[string]bool{
					"foo": true,
				},
				bp2buildModuleDoNotConvert: map[string]bool{
					"foo": true,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			bcc := &TestBazelConversionContext{
				omc: bazel.OtherModuleTestContext{
					Modules: []bazel.TestModuleInfo{
						test.module.TestModuleInfo,
					},
				},
			}

			shouldConvert := test.module.shouldConvertWithBp2build(bcc, bcc, test.module.TestModuleInfo, test.allowlist)
			if test.shouldConvert != shouldConvert {
				t.Errorf("Module shouldConvert expected to be: %v, but was: %v", test.shouldConvert, shouldConvert)
			}

			errorsMatch := true
			if len(test.expectedErrors) != len(bcc.errors) {
				errorsMatch = false
			} else {
				for i, err := range test.expectedErrors {
					if err != bcc.errors[i] {
						errorsMatch = false
					}
				}
			}
			if !errorsMatch {
				t.Errorf("Expected errors to be: %v, but were: %v", test.expectedErrors, bcc.errors)
			}
		})
	}
}
