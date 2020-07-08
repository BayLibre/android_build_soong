// Copyright 2020 Google Inc. All rights reserved.
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
	"path/filepath"

	"testing"
)

func TestSourceTreeModule(t *testing.T) {
	ctx, errs := setupSourceTreeTest(map[string]string{
		"dir1": `
			test_module {
				name: "a",
				release_version: "current",
			}
			test_module {
				name: "b",
				release_version: "30",
			}
			test_module {
				name: "c",
			}
		`,
	})
	FailIfErrored(t, errs)
	validateReleaseVersion(t, ctx, "a", "current")
	validateReleaseVersion(t, ctx, "b", "30")
	validateReleaseVersion(t, ctx, "c", "current")
}

func TestSourceTreeDependency(t *testing.T) {
	_, errs := setupSourceTreeTest(map[string]string{
		"dir1": `
		test_module {
			name: "a_dep_test",
			release_version: "30",
		}
		test_module {
			name: "b_dep_test",
			deps: ["a_dep_test"],
		}
		`,
	})
	FailIfNoMatchingErrors(t,
		"module \"b_dep_test\" variant \".*\": has a release version \"current\". "+
			"It must not depend on \"a_dep_test\" which has a different release version \"30\"",
		errs)
}

func TestDefaltableReleaseVersionProperty(t *testing.T) {
	ctx, errs := setupSourceTreeTest(map[string]string{
		"dir1": `
		defaults {
			name: "defaults",
			release_version: "30",
		}
		test_module {
			name: "a_with_default",
			defaults: ["defaults"],
		}
		`,
	})
	FailIfErrored(t, errs)
	validateReleaseVersion(t, ctx, "a_with_default", "30")
}

func setupSourceTreeTest(bps map[string]string) (ctx *TestContext, errs []error) {
	files := make(map[string][]byte, len(bps))
	for dir, text := range bps {
		files[filepath.Join(dir, "Android.bp")] = []byte(text)
	}

	config := TestArchConfig(buildDir, nil, "", files)

	ctx = NewTestArchContext()
	ctx.RegisterModuleType("test_module", defaultableTestModuleFactory)
	ctx.RegisterModuleType("defaults", defaultReleaseVersionFactory)
	ctx.PreArchMutators(RegisterDefaultsPreArchMutators)
	ctx.Register(config)

	_, errs = ctx.ParseBlueprintsFiles(".")
	if len(errs) > 0 {
		return ctx, errs
	}
	_, errs = ctx.PrepareBuildActions(config)
	return ctx, errs
}

type defaultableTestModuleProperties struct {
	Deps []string
}

type defaultableTestModule struct {
	ModuleBase
	DefaultableModuleBase

	properties defaultableTestModuleProperties
}

func (m *defaultableTestModule) DepsMutator(ctx BottomUpMutatorContext) {
	for _, d := range m.properties.Deps {
		ctx.AddDependency(ctx.Module(), dependencyTag{name: "test_dependency"}, d)
	}
}

func (m *defaultableTestModule) GenerateAndroidBuildActions(ModuleContext) {
}

func defaultableTestModuleFactory() Module {
	m := &defaultableTestModule{}
	m.AddProperties(&m.properties)
	InitAndroidArchModule(m, DeviceSupported, MultilibBoth)
	InitDefaultableModule(m)
	return m
}

type defaultsReleaseVersion struct {
	ModuleBase
	DefaultsModuleBase
}

func defaultReleaseVersionFactory() Module {
	defaults := &defaultsReleaseVersion{}
	defaults.AddProperties(&defaultableTestModuleProperties{})
	InitDefaultsModule(defaults)
	return defaults
}

func validateReleaseVersion(t *testing.T, ctx *TestContext, name string, expectedVersion string) {
	t.Helper()
	version := ctx.ModuleForTests(name, "android_arm64_armv8-a").Module().base().ReleaseVersion()
	if version != expectedVersion {
		t.Errorf("%q is expected to have release version %q but have %q",
			name, expectedVersion, version)
	}
}
