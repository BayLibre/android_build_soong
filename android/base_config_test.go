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
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

var baseConfigTestCases = []struct {
	// Inputs
	name           string
	bp             string
	vendorVars     map[string]map[string]string
	vendorVarTypes map[string]map[string]string

	// Outputs
	providers     map[string]baseConfigTestProvider
	expectedError string
}{
	{
		name: "Basic global configuration change",
		bp: `
my_module_type {
	name: "main_module",
	configuration: "my_base_config",
	my_string: select(soong_config_variable("my_namespace", "my_variable"), {
		"a": "a",
		default: "default",
	}),
	deps: [
		"sub_module",
	],
}

my_module_type {
	name: "sub_module",
	my_string: select(soong_config_variable("my_namespace", "my_variable"), {
		"a": "a",
		default: "default",
	}),
}

my_module_type {
	name: "non_configured_module",
	my_string: select(soong_config_variable("my_namespace", "my_variable"), {
		"a": "a",
		default: "default",
	}),
	deps: [
		"sub_module",
	],
}

base_configuration {
	name: "my_base_config",
	soong_config_variables: [
		"my_namespace:my_variable:string:a",
	],
}
		`,
		vendorVars: map[string]map[string]string{
			"my_namespace": {
				"my_variable": "b",
			},
		},
		providers: map[string]baseConfigTestProvider{
			"main_module:my_base_config_android_arm64_armv8-a": {
				my_string:             "a",
				transitive_my_strings: []string{"a"},
			},
			"non_configured_module:android_arm64_armv8-a": {
				my_string:             "default",
				transitive_my_strings: []string{"default"},
			},
		},
	},
}

func TestBaseConfig(t *testing.T) {
	for _, tc := range baseConfigTestCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := make(MockFS)
			if tc.bp != "" {
				fs["Android.bp"] = []byte(tc.bp)
			}
			fixtures := GroupFixturePreparers(
				PrepareForTestWithDefaults,
				PrepareForTestWithBaseConfig,
				PrepareForTestWithArchMutator,
				FixtureRegisterWithContext(func(ctx RegistrationContext) {
					ctx.RegisterModuleType("my_module_type", newBaseConfigTestModule)
				}),
				FixtureModifyProductVariables(func(variables FixtureProductVariables) {
					variables.VendorVars = tc.vendorVars
					variables.VendorVarTypes = tc.vendorVarTypes
				}),
				FixtureMergeMockFs(fs),
			)
			if tc.expectedError != "" {
				fixtures = fixtures.ExtendWithErrorHandler(FixtureExpectsOneErrorPattern(tc.expectedError))
			}
			result := fixtures.RunTest(t)

			if tc.expectedError == "" {
				for moduleNameAndVariant, expected := range tc.providers {
					moduleName, variant, ok := strings.Cut(moduleNameAndVariant, ":")
					if !ok {
						t.Errorf("Expected a module name and variant separated by :, found: %q", moduleNameAndVariant)
					}
					m := result.ModuleForTests(moduleName, variant)
					p, _ := OtherModuleProvider(result.testContext.OtherModuleProviderAdaptor(), m.Module(), baseConfigTestProviderKey)
					if !reflect.DeepEqual(p, expected) {
						t.Errorf("Expected:\n  %q\ngot:\n  %q", expected.String(), p.String())
					}
				}
			}
		})
	}
}

type baseConfigTestProvider struct {
	my_string             string
	transitive_my_strings []string
}

func (p *baseConfigTestProvider) String() string {
	return fmt.Sprintf(`baseConfigTestProvider {
	my_string: %s,
	transitive_my_strings: %v,
}`,
		p.my_string,
		p.transitive_my_strings,
	)
}

var baseConfigTestProviderKey = blueprint.NewProvider[baseConfigTestProvider]()

type baseConfigTestModuleProperties struct {
	My_string proptools.Configurable[string]
	Deps      []string
}

type baseConfigTestModule struct {
	ModuleBase
	DefaultableModuleBase
	MutableConfigurationModuleBase
	properties baseConfigTestModuleProperties
}

var _ Module = (*baseConfigTestModule)(nil)
var _ MutableConfigurationModule = (*baseConfigTestModule)(nil)

func newBaseConfigTestModule() Module {
	m := &baseConfigTestModule{}
	m.AddProperties(&m.properties)
	InitAndroidArchModule(m, HostAndDeviceSupported, MultilibFirst)
	InitDefaultableModule(m)
	InitMutableConfigurationModule(m)
	return m
}

func (p *baseConfigTestModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	var transitiveMyStrings []string
	ctx.VisitDirectDepsWithTag(baseConfigTestModuleDependencyTagInstance, func(m Module) {
		provider, ok := OtherModuleProvider(ctx, m, baseConfigTestProviderKey)
		if !ok {
			ctx.ModuleErrorf("Expected deps to have the correct provider")
			return
		}
		transitiveMyStrings = append(transitiveMyStrings, provider.my_string)
		transitiveMyStrings = append(transitiveMyStrings, provider.transitive_my_strings...)
	})
	transitiveMyStrings = SortedUniqueStrings(transitiveMyStrings)
	SetProvider(ctx, baseConfigTestProviderKey, baseConfigTestProvider{
		my_string:             p.properties.My_string.GetOrDefault(ctx, ""),
		transitive_my_strings: transitiveMyStrings,
	})
}

type baseConfigTestModuleDependencyTag struct {
	blueprint.BaseDependencyTag
}

var baseConfigTestModuleDependencyTagInstance = baseConfigTestModuleDependencyTag{}

func (p *baseConfigTestModule) DepsMutator(ctx BottomUpMutatorContext) {
	// Using AddVariationDependencies instead of AddDependency because AddDependency will silently
	// redirect you to using a different variant if it's the only variant that exists, which we don't want.
	// AddVariationDependencies will error out in that case.
	ctx.AddVariationDependencies(nil, baseConfigTestModuleDependencyTagInstance, p.properties.Deps...)
}
