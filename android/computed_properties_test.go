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

type computedPropertiesTestModule struct {
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
		Nested  *struct {
			String string
			PtrInt *int64
		}
	}
}

func (d computedPropertiesTestModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	ctx.Build(pctx, BuildParams{
		Rule:   Touch,
		Output: PathForModuleOut(ctx, "out"),
	})
}

var derivationTestFixture = FixtureRegisterWithContext(func(ctx RegistrationContext) {
	ctx.RegisterModuleType("test", func() Module {
		module := &computedPropertiesTestModule{}
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
		assertions func(module *computedPropertiesTestModule)
	}{
		{"zeroes",
			`test { computed: "def compute():\n\treturn {}"}`,
			func(m *computedPropertiesTestModule) {
				if m == nil {
					t.Errorf("No failure expected")
				}
			},
		}, {"zeroes on nested",
			`test { computed: "def compute():\n\treturn { 'nested' : {}}"}`,
			func(m *computedPropertiesTestModule) {
				if m == nil {
					t.Errorf("No failure expected")
				}
			},
		}, {
			"computed *int",
			`test { computed: "def compute():\n\treturn {'ptrInt': 20 - 30}"}`,
			func(m *computedPropertiesTestModule) {
				AssertIntEquals(t, "ptrInt", -10, int(*m.properties.PtrInt))
			},
		}, {
			"computed []int",
			`test { computed: "def compute():\n\treturn {'ints': [100, 200]}"}`,
			func(m *computedPropertiesTestModule) {
				AssertDeepEquals(t, "ints", []int64{100, 200}, m.properties.Ints)
			},
		}, {
			"computed bool",
			`test { computed: "def compute():\n\treturn {'bool': 1 == 1 and True}" }`,
			func(m *computedPropertiesTestModule) {
				AssertBoolEquals(t, "bool", true, m.properties.Bool)
			},
		}, {
			"computed *bool",
			`test { computed: "def compute():\n\treturn {'ptrBool': 1 < 2}"}`,
			func(m *computedPropertiesTestModule) {
				AssertBoolEquals(t, "ptrBool", true, *m.properties.PtrBool)
			},
		}, {
			"computed []bool",
			`test { computed: "def compute():\n\treturn {'bools': [True, False]}"}`,
			func(m *computedPropertiesTestModule) {
				AssertDeepEquals(t, "bools", []bool{true, false}, m.properties.Bools)
			},
		}, {
			"computed string",
			`test { computed: "def compute():\n\treturn {'string': 'abc'.upper()}" }`,
			func(m *computedPropertiesTestModule) {
				AssertStringEquals(t, "string", "ABC", m.properties.String)
				AssertStringEquals(t, "string", "ABC", *m.bizarroProperties.String)
			},
		}, {
			"computed *string",
			`test { computed: "def compute():\n\treturn {'ptrString': 'abc'.upper()}" }`,
			func(m *computedPropertiesTestModule) {
				AssertStringEquals(t, "ptrString", "ABC", *m.properties.PtrString)
				AssertStringEquals(t, "ptrString", "ABC", m.bizarroProperties.PtrString)
			},
		}, {
			"computed []string",
			`test { computed: "def compute():\n\treturn {'strings': ['a' + 'b']}" }`,
			func(m *computedPropertiesTestModule) {
				expected := []string{"ab"}
				if !reflect.DeepEqual(expected, m.properties.Strings) {
					t.Errorf("expected %v but was %v", expected, m.properties.Strings)
				}
			},
		}, {
			"computed [][]string",
			`test { computed: "def compute():\n\treturn {'stringss': [['a'], ['b', 'c']]}" }`,
			func(m *computedPropertiesTestModule) {
				expected := [][]string{{"a"}, {"b", "c"}}
				if !reflect.DeepEqual(expected, m.properties.Stringss) {
					t.Errorf("expected %v but was %v", expected, m.properties.Stringss)
				}
			},
		}, {
			"computed []*string",
			`test { computed: "def compute():\n\treturn {'ptrStrings': ['a', 'b']}" }`,
			func(m *computedPropertiesTestModule) {
				AssertIntEquals(t, "len(PtrStrings)", 2, len(m.properties.PtrStrings))
				AssertStringEquals(t, "PtrStrings[0]", "a", *m.properties.PtrStrings[0])
				AssertStringEquals(t, "PtrStrings[1]", "b", *m.properties.PtrStrings[1])
			},
		}, {
			"computed shared nested string",
			`test { computed: "def compute():\n\treturn {'nested': {'string': 'b'}}" }`,
			func(m *computedPropertiesTestModule) {
				AssertStringEquals(t, "Nested.String", "b", m.properties.Nested.String)
				AssertStringEquals(t, "Nested.String", "b", m.duplicateProperties.Nested.String)
			},
		}, {
			"computed disjoint nested properties",
			`test { computed: "def compute():\n\treturn {'nested': {'bool': True, 'ptrInt': 5}}" }`,
			func(m *computedPropertiesTestModule) {
				AssertBoolEquals(t, "Nested.Bool", true, m.properties.Nested.Bool)
				AssertIntEquals(t, "Nested.PtrInt", 5, int(*m.duplicateProperties.Nested.PtrInt))
			},
		},
	}

	for _, test := range successes {
		t.Run(test.name, func(t *testing.T) {
			result := derivationTestFixture.RunTestWithBp(t, test.bp)
			module := result.Module("m", "").(*computedPropertiesTestModule)
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
			`test { string: "a", computed: "def compute():\n\treturn {'string': 'a'}"}`,
			//`test { string: "", computed: "{'string': 'a'}"}`,
			// ^ won't create a collision unfortunately because "" is a Zero value
			`m\.string is already set`,
		}, {
			"bool collision",
			`test { bool: true, computed: "def compute():\n\treturn {'bool': True}"}`,
			//`test { bool: false, computed: "{'bool': True}"}`,
			// ^ won't create a collision unfortunately because false is a Zero value
			`m\.bool is already set`,
		}, {
			"list collision",
			`test { strings: [], computed: "def compute():\n\treturn {'strings': []}"}`,
			`m\.strings is already set`,
		}, {
			"pointer collision",
			`test { ptrInt: 0, computed: "def compute():\n\treturn {'ptrInt': 0}"}`,
			`m\.ptrInt is already set`,
		}, {
			"blueprint mutated",
			`test { computed: "def compute():\n\treturn {'untouchable': 'b'}"}`,
			`m\.untouchable is tagged blueprint:"mutated"`,
		}, {
			"unknown property",
			`test {computed: "def compute():\n\treturn {'what': 0}"}`,
			`unused:\n\tm\.what`,
		}, {
			"unknown nested property",
			`test {computed: "def compute():\n\treturn {'nested': {'what': 0}}"}`,
			`unused:\n\tm\.nested\.what`,
		}, {
			"unknown property and nested property",
			`test {computed: "def compute():\n\treturn {'what': True, 'nested': {'what': 0}}"}`,
			`unused:\n\tm\.nested\.what\n\tm\.what`,
		}, {
			"unknown and known nested property",
			`test {computed: "def compute():\n\treturn {'nested': {'who': 0, 'String': 'a', 'what': False}}"}`,
			`unused:\n\tm\.nested\.what\n\tm\.nested\.who`,
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
