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
			dummy_module {
				name: "a",
				release_version: "current",
			}
			dummy_module {
				name: "b",
				release_version: "30",
			}
			dummy_module {
				name: "c",
			}
		`,
	})
	FailIfErrored(t, errs)

	getModuleForReleaseVersion(ctx, "a", "")
	getModuleForReleaseVersion(ctx, "b", "30")
	getModuleForReleaseVersion(ctx, "c", "")
}

func TestSourceTreeDependency(t *testing.T) {
	_, errs := setupSourceTreeTest(map[string]string{
		"dir1": `
		dummy_module {
			name: "a_version_test",
			release_version: "30",
		}
		dummy_module {
			name: "b_version_test",
			deps: ["a_version_test"],
		}
		dummy_module {
			name: "c_version_test",
			deps: ["b_version_test"],
			release_version: "current",
		}
		`,
	})
	FailIfNoMatchingErrors(t,
		"dependency \"a_version_test\" of \"b_version_test\" missing variant",
		errs)
}

func TestDefaltableReleaseVersionProperty(t *testing.T) {
	ctx, errs := setupSourceTreeTest(map[string]string{
		"dir1": `
		defaults {
			name: "defaults",
			release_version: "30",
		}
		dummy_module {
			name: "a_with_default",
			defaults: ["defaults"],
		}
		`,
	})
	FailIfErrored(t, errs)
	getModuleForReleaseVersion(ctx, "a_with_default", "30")
}

func setupSourceTreeTest(bps map[string]string) (ctx *TestContext, errs []error) {
	files := make(map[string][]byte, len(bps))
	for dir, text := range bps {
		files[filepath.Join(dir, "Android.bp")] = []byte(text)
	}

	config := TestArchConfig(buildDir, nil, "", files)

	ctx = NewTestArchContext()
	ctx.RegisterModuleType("dummy_module", dummyModuleFactory)
	ctx.RegisterModuleType("defaults", defaultReleaseVersionFactory)
	ctx.PreArchMutators(RegisterDefaultsPreArchMutators)
	ctx.PreDepsMutators(RegisterSourceTreeMutators)
	ctx.Register(config)

	_, errs = ctx.ParseBlueprintsFiles(".")
	if len(errs) > 0 {
		return ctx, errs
	}
	_, errs = ctx.PrepareBuildActions(config)
	return ctx, errs
}

type dummyModuleProperties struct {
	Deps []string
}

type dummyModule struct {
	ModuleBase
	DefaultableModuleBase

	properties dummyModuleProperties
}

func (m *dummyModule) DepsMutator(ctx BottomUpMutatorContext) {
	for _, d := range m.properties.Deps {
		ctx.AddDependency(ctx.Module(), dependencyTag{name: "test_dependency"}, d)
	}
}

func (m *dummyModule) GenerateAndroidBuildActions(ModuleContext) {
}

func dummyModuleFactory() Module {
	m := &dummyModule{}
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
	defaults.AddProperties(&dummyModuleProperties{})
	InitDefaultsModule(defaults)
	return defaults
}

func getModuleForReleaseVersion(ctx *TestContext, name string, version string) Module {
	variation := "android_arm64_armv8-a"
	if version != "" {
		variation += "_" + version
	}
	return ctx.ModuleForTests(name, variation).Module()
}
