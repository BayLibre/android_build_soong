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

	"go.starlark.net/starlark"
)

type derivationsTestModule struct {
	ModuleBase
	properties struct {
		String      string
		PtrString   *string
		Strings     []string
		Stringss    [][]string
		PtrStrings  []*string
		PtrInt      *int64
		Ints        []int64
		Bool        bool
		PtrBool     *bool
		Bools       []bool
		Untouchable string `blueprint:"mutated"`
		Nested      struct {
			String string
			Bool   bool
		}
	}
	bizarroProperties struct {
		String    *string
		PtrString string
	}
	duplicateProperties struct {
		Strings []string
		Nested  struct {
			String string
			PtrInt *int64
		}
	}
}

func (d derivationsTestModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	ctx.Build(pctx, BuildParams{
		Rule:   Touch,
		Output: PathForModuleOut(ctx, "out"),
	})
}

var derivationTestFixture = FixtureRegisterWithContext(func(ctx RegistrationContext) {
	ctx.RegisterModuleType("test", func() Module {
		module := &derivationsTestModule{}
		module.nameProperties.Name = stringPtr("m")
		InitAndroidModule(module)
		module.AddProperties(&module.properties, &module.bizarroProperties, &module.duplicateProperties)
		return module
	})
	ctx.PreArchMutators(RegisterDerivedPropertiesPreArchMutator)
})

func TestRegisterDerivedPropertiesPreArchMutator(t *testing.T) {
	successes := []struct {
		name       string
		bp         string
		assertions func(module *derivationsTestModule)
	}{
		{"zeroes",
			`test { derived: "def derive():\n\treturn {}"}`,
			func(m *derivationsTestModule) {
				if m == nil {
					t.Errorf("No failure expected")
				}
			},
		}, {"zeroes on nested",
			`test { derived: "def derive():\n\treturn { 'nested' : {}}"}`,
			func(m *derivationsTestModule) {
				if m == nil {
					t.Errorf("No failure expected")
				}
			},
		}, {
			"derived *int",
			`test { derived: "def derive():\n\treturn {'ptrInt': 20 - 30}"}`,
			func(m *derivationsTestModule) {
				AssertIntEquals(t, "ptrInt", -10, int(*m.properties.PtrInt))
			},
		}, {
			"derived []int",
			`test { derived: "def derive():\n\treturn {'ints': [100, 200]}"}`,
			func(m *derivationsTestModule) {
				AssertDeepEquals(t, "ints", []int64{100, 200}, m.properties.Ints)
			},
		}, {
			"derived bool",
			`test { derived: "def derive():\n\treturn {'bool': 1 == 1 and True}" }`,
			func(m *derivationsTestModule) {
				AssertBoolEquals(t, "bool", true, m.properties.Bool)
			},
		}, {
			"derived *bool",
			`test { derived: "def derive():\n\treturn {'ptrBool': 1 < 2}"}`,
			func(m *derivationsTestModule) {
				AssertBoolEquals(t, "ptrBool", true, *m.properties.PtrBool)
			},
		}, {
			"derived []bool",
			`test { derived: "def derive():\n\treturn {'bools': [True, False]}"}`,
			func(m *derivationsTestModule) {
				AssertDeepEquals(t, "bools", []bool{true, false}, m.properties.Bools)
			},
		}, {
			"derived string",
			`test { derived: "def derive():\n\treturn {'string': 'abc'.upper()}" }`,
			func(m *derivationsTestModule) {
				AssertStringEquals(t, "string", "ABC", m.properties.String)
				AssertStringEquals(t, "string", "ABC", *m.bizarroProperties.String)
			},
		}, {
			"derived *string",
			`test { derived: "def derive():\n\treturn {'ptrString': 'abc'.upper()}" }`,
			func(m *derivationsTestModule) {
				AssertStringEquals(t, "ptrString", "ABC", *m.properties.PtrString)
				AssertStringEquals(t, "ptrString", "ABC", m.bizarroProperties.PtrString)
			},
		}, {
			"derived []string",
			`test { derived: "def derive():\n\treturn {'strings': ['a' + 'b']}" }`,
			func(m *derivationsTestModule) {
				expected := []string{"ab"}
				if !reflect.DeepEqual(expected, m.properties.Strings) {
					t.Errorf("expected %v but was %v", expected, m.properties.Strings)
				}
			},
		}, {
			"derived [][]string",
			`test { derived: "def derive():\n\treturn {'stringss': [['a'], ['b', 'c']]}" }`,
			func(m *derivationsTestModule) {
				expected := [][]string{{"a"}, {"b", "c"}}
				if !reflect.DeepEqual(expected, m.properties.Stringss) {
					t.Errorf("expected %v but was %v", expected, m.properties.Stringss)
				}
			},
		}, {
			"derived []*string",
			`test { derived: "def derive():\n\treturn {'ptrStrings': ['a', 'b']}" }`,
			func(m *derivationsTestModule) {
				AssertIntEquals(t, "len(PtrStrings)", 2, len(m.properties.PtrStrings))
				AssertStringEquals(t, "PtrStrings[0]", "a", *m.properties.PtrStrings[0])
				AssertStringEquals(t, "PtrStrings[1]", "b", *m.properties.PtrStrings[1])
			},
		}, {
			"derived shared nested string",
			`test { derived: "def derive():\n\treturn {'nested': {'string': 'b'}}" }`,
			func(m *derivationsTestModule) {
				AssertStringEquals(t, "Nested.String", "b", m.properties.Nested.String)
				AssertStringEquals(t, "Nested.String", "b", m.duplicateProperties.Nested.String)
			},
		}, {
			"derived disjoint nested properties",
			`test { derived: "def derive():\n\treturn {'nested': {'bool': True, 'ptrInt': 5}}" }`,
			func(m *derivationsTestModule) {
				AssertBoolEquals(t, "Nested.Bool", true, m.properties.Nested.Bool)
				AssertIntEquals(t, "Nested.PtrInt", 5, int(*m.duplicateProperties.Nested.PtrInt))
			},
		},
	}

	for _, test := range successes {
		t.Run(test.name, func(t *testing.T) {
			result := derivationTestFixture.RunTestWithBp(t, test.bp)
			module := result.Module("m", "").(*derivationsTestModule)
			test.assertions(module)
		})
	}
}

func TestRegisterDerivedPropertiesPreArchMutator_Errors(t *testing.T) {
	failures := []struct {
		name           string
		bp             string
		failureMessage string
	}{
		{
			"string collision",
			`test { string: "a", derived: "def derive():\n\treturn {'string': 'a'}"}`,
			//`test { string: "", derived: "{'string': 'a'}"}`,
			// ^ won't create a collision unfortunately because "" is a Zero value
			`already set "m\.String" = "a"`,
		}, {
			"bool collision",
			`test { bool: true, derived: "def derive():\n\treturn {'bool': True}"}`,
			//`test { bool: false, derived: "{'bool': True}"}`,
			// ^ won't create a collision unfortunately because false is a Zero value
			`already set "m\.Bool" = true`,
		}, {
			"list collision",
			`test { strings: [], derived: "def derive():\n\treturn {'strings': []}"}`,
			`already set "m\.Strings" = \[\]`,
		}, {
			"pointer collision",
			`test { ptrInt: 0, derived: "def derive():\n\treturn {'ptrInt': 0}"}`,
			`already set "\*m\.PtrInt" = 0`,
		}, {
			"blueprint mutated",
			`test { derived: "def derive():\n\treturn {'untouchable': 'b'}"}`,
			`"m\.Untouchable" is tagged blueprint:"mutated"`,
		}, {
			"unknown property",
			`test {derived: "def derive():\n\treturn {'what': 0}"}`,
			`unused "m\.what":`,
		}, {
			"unknown nested property",
			`test {derived: "def derive():\n\treturn {'nested': {'what': 0}}"}`,
			`unused "m\.nested\.what":`,
		}, {
			"unknown property and nested property",
			`test {derived: "def derive():\n\treturn {'what': True, 'nested': {'what': 0}}"}`,
			`unused "m\.nested\.what, m\.what":`,
		}, {
			"unknown and known nested property",
			`test {derived: "def derive():\n\treturn {'nested': {'who': 0, 'String': 'a', 'what': False}}"}`,
			`unused "m\.nested\.what, m\.nested\.who":`,
		},
	}

	for _, test := range failures {
		t.Run(test.name, func(t *testing.T) {
			derivationTestFixture.
				ExtendWithErrorHandler(FixtureExpectsAtLeastOneErrorMatchingPattern(test.failureMessage)).
				RunTestWithBp(t, test.bp)
		})
	}
}

func Test_asValue(t *testing.T) {
	type args struct {
		starlarkValue starlark.Value
		t             reflect.Type
	}
	tests := []struct {
		name string
		args args
		want reflect.Value
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := asValue(tt.args.starlarkValue, tt.args.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_evaluateDerivations(t *testing.T) {
	type args struct {
		ctx    BaseMutatorContext
		script string
	}
	tests := []struct {
		name string
		args args
		want *starlark.Dict
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluateDerivations(tt.args.ctx, &tt.args.script); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("evaluateDerivations() = %v, want %v", got, tt.want)
			}
		})
	}
}
