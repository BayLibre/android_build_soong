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

type vendorTestModule struct {
	ModuleBase
	props testModuleProperties
}

type testModuleProperties struct {
	Cflags []string
}

func testModuleFactory() Module {
	m := &vendorTestModule{}
	m.AddProperties(&m.props)
	InitAndroidModule(m)
	return m
}

func (t vendorTestModule) GenerateAndroidBuildActions(ModuleContext) {}

func TestVendorModule(t *testing.T) {
	bp := `
		soong_vendor_vars_import {
			from: "VendorVars.bp",
			module_types: ["acme_test_defaults"],
		}
	
		acme_test_defaults {
			name: "foo",
			cflags: ["-DGENERIC"],
			vendor_variables: {
				board: {
					soc_a: {
						cflags: ["-DSOC_A"],
					},
					soc_b: {
						cflags: ["-DSOC_B"],
					},
				},
				feature1: {
					cflags: ["-DFEATURE1"],
				},
				feature2: {
					cflags: ["-DFEATURE2"],
				},
				feature3: {
					cflags: ["-DFEATURE3"],
				},
			},
		}
	`

	vendorVars := `
		soong_vendor_vars_module_type {
			name: "acme_test_defaults",
			module_type: "test_defaults",
			config_namespace: "acme",
			variables: ["board", "feature1", "feature2", "feature3"],
			properties: ["cflags", "srcs"],
		}

		soong_vendor_vars_string_variable {
			name: "board",
			values: ["soc_a", "soc_b"],
		}

		soong_vendor_vars_bool_variable {
			name: "feature1",
		}

		soong_vendor_vars_bool_variable {
			name: "feature2",
		}

		soong_vendor_vars_bool_variable {
			name: "feature3",
		}
	`

	fs := map[string][]byte{
		"VendorVars.bp": []byte(vendorVars),
	}

	config := TestConfig(buildDir, nil, bp, fs)

	config.TestProductVariables.VendorVars = map[string]map[string]string{
		"acme": map[string]string{
			"board":    "soc_a",
			"feature1": "true",
			"feature2": "false",
			// FEATURE3 unset
		},
	}

	ctx := NewTestContext()
	ctx.RegisterModuleType("soong_vendor_vars_import", vendorVarsImportFactory)
	ctx.RegisterModuleType("test_defaults", testModuleFactory)
	ctx.Register(config)

	_, errs := ctx.ParseBlueprintsFiles("Android.bp")
	FailIfErrored(t, errs)
	_, errs = ctx.PrepareBuildActions(config)
	FailIfErrored(t, errs)

	foo := ctx.ModuleForTests("foo", "").Module().(*vendorTestModule)
	if g, w := foo.props.Cflags, []string{"-DGENERIC", "-DSOC_A", "-DFEATURE1"}; !reflect.DeepEqual(g, w) {
		t.Errorf("wanted foo cflags %q, got %q", w, g)
	}
}

func Test_canonicalizeVendorVarToProperty(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "lowercase",
			arg:  "board",
			want: "board",
		},
		{
			name: "uppercase",
			arg:  "BOARD",
			want: "BOARD",
		},
		{
			name: "numbers",
			arg:  "BOARD123",
			want: "BOARD123",
		},
		{
			name: "underscore",
			arg:  "TARGET_BOARD",
			want: "TARGET_BOARD",
		},
		{
			name: "dash",
			arg:  "TARGET-BOARD",
			want: "TARGET_BOARD",
		},
		{
			name: "unicode",
			arg:  "boardλ",
			want: "board_",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canonicalizeToProperty(tt.arg); got != tt.want {
				t.Errorf("canonicalizeToProperty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_typeForPropertyFromPropertyStruct(t *testing.T) {
	tests := []struct {
		name     string
		ps       interface{}
		property string
		want     string
	}{
		{
			name: "string",
			ps: struct {
				A string
			}{},
			property: "a",
			want:     "string",
		},
		{
			name: "list",
			ps: struct {
				A []string
			}{},
			property: "a",
			want:     "[]string",
		},
		{
			name: "missing",
			ps: struct {
				A []string
			}{},
			property: "b",
			want:     "",
		},
		{
			name: "nested",
			ps: struct {
				A struct {
					B string
				}
			}{},
			property: "a.b",
			want:     "string",
		},
		{
			name: "missing nested",
			ps: struct {
				A struct {
					B string
				}
			}{},
			property: "a.c",
			want:     "",
		},
		{
			name: "not a struct",
			ps: struct {
				A string
			}{},
			property: "a.b",
			want:     "",
		},
		{
			name: "nested pointer",
			ps: struct {
				A *struct {
					B string
				}
			}{},
			property: "a.b",
			want:     "string",
		},
		{
			name: "nested interface",
			ps: struct {
				A interface{}
			}{
				A: struct {
					B string
				}{},
			},
			property: "a.b",
			want:     "string",
		},
		{
			name: "nested interface pointer",
			ps: struct {
				A interface{}
			}{
				A: &struct {
					B string
				}{},
			},
			property: "a.b",
			want:     "string",
		},
		{
			name: "nested interface nil pointer",
			ps: struct {
				A interface{}
			}{
				A: (*struct {
					B string
				})(nil),
			},
			property: "a.b",
			want:     "string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := typeForPropertyFromPropertyStruct(tt.ps, tt.property)
			got := ""
			if typ != nil {
				got = typ.String()
			}
			if got != tt.want {
				t.Errorf("typeForPropertyFromPropertyStruct() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createAffectablePropertiesType(t *testing.T) {
	tests := []struct {
		name                 string
		affectableProperties []string
		factoryProps         interface{}
		want                 string
	}{
		{
			name:                 "string",
			affectableProperties: []string{"cflags"},
			factoryProps: struct {
				Cflags string
			}{},
			want: "*struct { Cflags string }",
		},
		{
			name:                 "list",
			affectableProperties: []string{"cflags"},
			factoryProps: struct {
				Cflags []string
			}{},
			want: "*struct { Cflags []string }",
		},
		{
			name:                 "string pointer",
			affectableProperties: []string{"cflags"},
			factoryProps: struct {
				Cflags *string
			}{},
			want: "*struct { Cflags *string }",
		},
		{
			name:                 "subset",
			affectableProperties: []string{"cflags"},
			factoryProps: struct {
				Cflags  string
				Ldflags string
			}{},
			want: "*struct { Cflags string }",
		},
		{
			name:                 "none",
			affectableProperties: []string{"cflags"},
			factoryProps: struct {
				Ldflags string
			}{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := createAffectablePropertiesType(tt.affectableProperties, []interface{}{tt.factoryProps})
			got := ""
			if typ != nil {
				got = typ.String()
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createAffectablePropertiesType() = %v, want %v", got, tt.want)
			}
		})
	}
}
