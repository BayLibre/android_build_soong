// Copyright 2024 Google Inc. All rights reserved.
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
	"testing"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func TestSelects(t *testing.T) {
	bp := `
my_module_type {
	name: "foo",
	my_string_list: select soong_config_variable: "my_namespace" "my_variable" {
		"a": ["a.cpp"],
		"b": ["b.cpp"],
		default: ["c.cpp"],
	},
}
`
	result := GroupFixturePreparers(
		PrepareForTestWithFilegroup,
		FixtureRegisterWithContext(func(ctx RegistrationContext) {
			ctx.RegisterModuleType("my_module_type", newSelectsMockModule)
		}),
	).RunTestWithBp(t, bp)
	m := result.ModuleForTests("foo", "")

	p, _ := OtherModuleProvider[selectsTestProvider](result.testContext.OtherModuleProviderAdaptor(), m.Module(), selectsTestProviderKey)
	if p.my_string_list[0] != "c.cpp" {
		t.Errorf("Expected \"c.cpp\", got %q", p.my_string_list[0])
	}
}

func TestSelects2(t *testing.T) {
	bp := `
my_module_type {
	name: "foo",
	my_string_list: select soong_config_variable: "my_namespace" "my_variable" {
		"a": ["a.cpp"],
		"b": ["b.cpp"],
		default: ["c.cpp"],
	},
}
`
	result := GroupFixturePreparers(
		PrepareForTestWithFilegroup,
		FixtureRegisterWithContext(func(ctx RegistrationContext) {
			ctx.RegisterModuleType("my_module_type", newSelectsMockModule)
		}),
		FixtureModifyProductVariables(func(variables FixtureProductVariables) {
			variables.VendorVars = map[string]map[string]string{
				"my_namespace": {
					"my_variable": "a",
				},
			}
		}),
	).RunTestWithBp(t, bp)
	m := result.ModuleForTests("foo", "")

	p, _ := OtherModuleProvider[selectsTestProvider](result.testContext.OtherModuleProviderAdaptor(), m.Module(), selectsTestProviderKey)
	if p.my_string_list[0] != "a.cpp" {
		t.Errorf("Expected \"a.cpp\", got %q", p.my_string_list[0])
	}
}

type selectsTestProvider struct {
	my_string_list []string
}

var selectsTestProviderKey = blueprint.NewProvider[selectsTestProvider]()

type selectsMockModuleProperties struct {
	My_string_list proptools.Configurable[[]string]
}

type selectsMockModule struct {
	ModuleBase
	DefaultableModuleBase
	properties selectsMockModuleProperties
}

func (p *selectsMockModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	SetProvider[selectsTestProvider](ctx, selectsTestProviderKey, selectsTestProvider{
		my_string_list: *p.properties.My_string_list.Evaluate(ctx),
	})
}

func newSelectsMockModule() Module {
	m := &selectsMockModule{}
	m.AddProperties(&m.properties)
	InitAndroidArchModule(m, HostAndDeviceSupported, MultilibCommon)
	InitDefaultableModule(m)
	return m
}
