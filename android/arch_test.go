// Copyright 2016 Google Inc. All rights reserved.
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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/google/blueprint"
)

var buildDir string

func TestMain(m *testing.M) {
	flag.Parse()
	var err error

	buildDir, err = ioutil.TempDir("", "soong_test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}

	ret := m.Run()

	os.RemoveAll(buildDir)
	os.Exit(ret)
}

func archTest(t *testing.T, test archTestCase) {
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
		if a.Arch().ArchType != test.arch[i] {
			t.Errorf("incorrect arch for %d, expected %s got %s", i, test.arch[i], a.Arch().ArchType)
		}

		i++
	})
	if i != len(test.modules) {
		t.Fatalf("expected %d modules, got %d", len(test.modules), i)
	}
}

func TestArchMerge(t *testing.T) {
	for _, test := range archTestCases {
		archTest(t, test)
	}
}

type archTestCase struct {
	file    string
	modules []archTestModuleProperties
	errs    bool
	arch    []ArchType
}

var archTestCases = []archTestCase{
	{
		file: `
			arch_test_module {
				list: ["foo"],
				string: "foo",
				nested: {
					list: ["nested_foo"],
					string: "nested_foo",
				},
				prepend: {
					list: ["prepend_foo"],
					string: "prepend_foo",
				},
				arch: {
					arm: {
						list: ["bar"],
						string: "bar",
						nested: {
							list: ["nested_bar"],
							string: "nested_bar",
						},
						prepend: {
							list: ["prepend_bar"],
							string: "prepend_bar",
						},
					},
					arm64: {
						list: ["baz"],
						string: "baz",
						nested: {
							list: ["nested_baz"],
							string: "nested_baz",
						},
						prepend: {
							list: ["prepend_baz"],
							string: "prepend_baz",
						},
					},
				},
			}
		`,
		modules: []archTestModuleProperties{
			{
				List:   []string{"foo", "baz"},
				String: "foobaz",
				Nested: struct {
					List   []string `android:"arch_variant"`
					String string   `android:"arch_variant"`
				}{
					List:   []string{"nested_foo", "nested_baz"},
					String: "nested_foonested_baz",
				},
				Prepend: struct {
					List   []string `android:"arch_variant,variant_prepend"`
					String string   `android:"arch_variant,variant_prepend"`
				}{
					List:   []string{"prepend_baz", "prepend_foo"},
					String: "prepend_bazprepend_foo",
				},
			},
			{
				List:   []string{"foo", "bar"},
				String: "foobar",
				Nested: struct {
					List   []string `android:"arch_variant"`
					String string   `android:"arch_variant"`
				}{
					List:   []string{"nested_foo", "nested_bar"},
					String: "nested_foonested_bar",
				},
				Prepend: struct {
					List   []string `android:"arch_variant,variant_prepend"`
					String string   `android:"arch_variant,variant_prepend"`
				}{
					List:   []string{"prepend_bar", "prepend_foo"},
					String: "prepend_barprepend_foo",
				},
			},
		},
		arch: []ArchType{
			Arm64,
			Arm,
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
