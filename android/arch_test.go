// Copyright 2015 Google Inc. All rights reserved.
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
	"android/soong"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/google/blueprint"
)

func TestArchMerge(t *testing.T) {
	for _, test := range testCases {
		buildDir, err := ioutil.TempDir("", "soong_arch_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(buildDir)

		config, err := NewConfig(".", buildDir)
		if err != nil {
			t.Fatal(err)
		}

		ctx := soong.NewContext()
		ctx.RegisterModuleType("arch_test_module", newArchTestModule)
		ctx.MockFileSystem(map[string][]byte{
			"Android.bp": []byte(test.file),
		})
		_, errs := ctx.ParseBlueprintsFiles("Android.bp")
		if errs != nil && !test.errs {
			t.Errorf("Unexpected errors")
			for _, err := range errs {
				t.Errorf("  %s", err.Error())
			}
			t.FailNow()
		} else if errs == nil && test.errs {
			t.Fatal("Missing expected error")
		}

		errs = ctx.ResolveDependencies(config)
		if len(errs) > 0 {
			t.Errorf("unexpected dep errors:")
			for _, err := range errs {
				t.Errorf("  %s", err.Error())
			}
			t.FailNow()
		}
		i := 0
		ctx.VisitAllModules(func(module blueprint.Module) {
			a := module.(*archTestModule)
			if !reflect.DeepEqual(a.properties, test.modules[i]) {
				t.Errorf("module %d does not match", i)
				t.Errorf("  expected: %v", a.properties)
				t.Errorf("  got:      %v", test.modules[i])
			}
			i++
		})
		if i != len(test.modules) {
			t.Fatalf("expected %d modules, got %d", len(test.modules), i)
		}
	}
}

var testCases = []struct {
	file    string
	modules []archTestModuleProperties
	errs    bool
}{
	{
		file: `
			arch_test_module {
				list: ["foo"],
				arch: {
					arm: {
						list: ["bar"],
					},
					arm64: {
						list: ["baz"],
					},
				},
			}
		`,
		modules: []archTestModuleProperties{
			{
				List: []string{"foo", "baz"},
			},
			{
				List: []string{"foo", "bar"},
			},
		},
	},
}

func newArchTestModule() (blueprint.Module, []interface{}) {
	m := &archTestModule{}
	return InitAndroidArchModule(m, DeviceSupported, MultilibBoth, &m.properties)
}

type archTestModuleProperties struct {
	List   []string `android:"arch_variant"`
	String string   `android:"arch_variant"`
	Nested struct {
		List   []string `android:"arch_variant"`
		String string   `android:"arch_variant"`
	}
	Prepend struct {
		List   []string `android:"arch_variant,variant_prepend"`
		String string   `android:"arch_variant,variant_prepend"`
	}
}

type archTestModule struct {
	ModuleBase
	properties archTestModuleProperties
}

func (archTestModule) GenerateAndroidBuildActions(ModuleContext) {}
