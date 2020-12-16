// Copyright 2019 Google Inc. All rights reserved.
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
	"reflect"
	"testing"
)

type soongConfigTestDefaultsModuleProperties struct {
}

type soongConfigTestDefaultsModule struct {
	ModuleBase
	DefaultsModuleBase
}

func soongConfigTestDefaultsModuleFactory() Module {
	m := &soongConfigTestDefaultsModule{}
	m.AddProperties(&soongConfigTestModuleProperties{})
	InitDefaultsModule(m)
	return m
}

type soongConfigTestModule struct {
	ModuleBase
	DefaultableModuleBase
	props soongConfigTestModuleProperties
}

type soongConfigTestModuleProperties struct {
	Cflags []string
}

func soongConfigTestModuleFactory() Module {
	m := &soongConfigTestModule{}
	m.AddProperties(&m.props)
	InitAndroidModule(m)
	InitDefaultableModule(m)
	return m
}

func (t soongConfigTestModule) GenerateAndroidBuildActions(ModuleContext) {}

func TestSoongConfigModule(t *testing.T) {
	configBp := `
		soong_config_module_type {
			name: "acme_test",
			module_type: "test",
			config_namespace: "acme",
			variables: ["board", "feature1", "FEATURE3"],
			bool_variables: ["feature2"],
			value_variables: ["size"],
			properties: ["cflags", "srcs", "defaults"],
		}

		soong_config_string_variable {
			name: "board",
			values: ["soc_a", "soc_b"],
		}

		soong_config_bool_variable {
			name: "feature1",
		}

		soong_config_bool_variable {
			name: "FEATURE3",
		}
	`

	importBp := `
		soong_config_module_type_import {
			from: "SoongConfig.bp",
			module_types: ["acme_test"],
		}
	`

	bp := `
		test_defaults {
			name: "foo_defaults",
			cflags: ["DEFAULT"],
		}

		acme_test {
			name: "foo",
			cflags: ["-DGENERIC"],
			defaults: ["foo_defaults"],
			soong_config_variables: {
				board: {
					soc_a: {
						cflags: ["-DSOC_A"],
					},
					soc_b: {
						cflags: ["-DSOC_B"],
					},
				},
				size: {
					cflags: ["-DSIZE=%s"],
				},
				feature1: {
					cflags: ["-DFEATURE1"],
				},
				feature2: {
					cflags: ["-DFEATURE2"],
				},
				FEATURE3: {
					cflags: ["-DFEATURE3"],
				},
				conditions_default: {
					cflags: ["-DCONDITIONS_DEFAULT"],
				},
			},
		}

		test_defaults {
			name: "foo_defaults_a",
			cflags: ["DEFAULT_A"],
		}

		test_defaults {
			name: "foo_defaults_b",
			cflags: ["DEFAULT_B"],
		}

		acme_test {
			name: "foo_with_defaults",
			cflags: ["-DGENERIC"],
			defaults: ["foo_defaults"],
			soong_config_variables: {
				board: {
					soc_a: {
						cflags: ["-DSOC_A"],
						defaults: ["foo_defaults_a"],
					},
					soc_b: {
						cflags: ["-DSOC_B"],
						defaults: ["foo_defaults_b"],
					},
				},
				size: {
					cflags: ["-DSIZE=%s"],
				},
				feature1: {
					cflags: ["-DFEATURE1"],
				},
				feature2: {
					cflags: ["-DFEATURE2"],
				},
				FEATURE3: {
					cflags: ["-DFEATURE3"],
				},
				conditions_default: {
					cflags: ["-DCONDITIONS_DEFAULT"],
				},
			},
		}
    `

	run := func(t *testing.T, bp string, fs map[string][]byte) {
		testCases := []struct {
			name                     string
			config                   Config
			fooExpectedFlags         []string
			fooDefaultsExpectedFlags []string
		}{
			{
				name: "withValues",
				config: testConfigWithVendorVars(buildDir, bp, fs, map[string]map[string]string{
					"acme": map[string]string{
						"board":    "soc_a",
						"size":     "42",
						"feature1": "true",
						"feature2": "false",
						// FEATURE3 unset
					},
				}),
				fooExpectedFlags:         []string{"DEFAULT", "-DGENERIC", "-DSIZE=42", "-DSOC_A", "-DFEATURE1"},
				fooDefaultsExpectedFlags: []string{"DEFAULT_A", "DEFAULT", "-DGENERIC", "-DSIZE=42", "-DSOC_A", "-DFEATURE1"},
			},
			{
				name:                     "conditions_default",
				config:                   testConfigWithVendorVars(buildDir, bp, fs, map[string]map[string]string{}),
				fooExpectedFlags:         []string{"DEFAULT", "-DGENERIC", "-DCONDITIONS_DEFAULT"},
				fooDefaultsExpectedFlags: []string{"DEFAULT", "-DGENERIC", "-DCONDITIONS_DEFAULT"},
			},
		}

		for _, tc := range testCases {
			ctx := NewTestContext(tc.config)
			ctx.RegisterModuleType("soong_config_module_type_import", soongConfigModuleTypeImportFactory)
			ctx.RegisterModuleType("soong_config_module_type", soongConfigModuleTypeFactory)
			ctx.RegisterModuleType("soong_config_string_variable", soongConfigStringVariableDummyFactory)
			ctx.RegisterModuleType("soong_config_bool_variable", soongConfigBoolVariableDummyFactory)
			ctx.RegisterModuleType("test_defaults", soongConfigTestDefaultsModuleFactory)
			ctx.RegisterModuleType("test", soongConfigTestModuleFactory)
			ctx.PreArchMutators(RegisterDefaultsPreArchMutators)
			ctx.Register()

			_, errs := ctx.ParseBlueprintsFiles("Android.bp")
			FailIfErrored(t, errs)
			_, errs = ctx.PrepareBuildActions(tc.config)
			FailIfErrored(t, errs)

			foo := ctx.ModuleForTests("foo", "").Module().(*soongConfigTestModule)
			if g, w := foo.props.Cflags, tc.fooExpectedFlags; !reflect.DeepEqual(g, w) {
				t.Errorf("%s: wanted foo cflags %q, got %q", tc.name, w, g)
			}

			fooDefaults := ctx.ModuleForTests("foo_with_defaults", "").Module().(*soongConfigTestModule)
			if g, w := fooDefaults.props.Cflags, tc.fooDefaultsExpectedFlags; !reflect.DeepEqual(g, w) {
				t.Errorf("%s: wanted foo_with_defaults cflags %q, got %q", tc.name, w, g)
			}
		}

	}

	t.Run("single file", func(t *testing.T) {
		run(t, configBp+bp, nil)
	})

	t.Run("import", func(t *testing.T) {
		run(t, importBp+bp, map[string][]byte{
			"SoongConfig.bp": []byte(configBp),
		})
	})
}

func testConfigWithVendorVars(buildDir, bp string, fs map[string][]byte, vendorVars map[string]map[string]string) Config {
	config := TestConfig(buildDir, nil, bp, fs)

	config.TestProductVariables.VendorVars = vendorVars

	return config
}
