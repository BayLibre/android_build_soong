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
	"fmt"
	"reflect"
	"testing"
)

type nestedComputedPropertiesForTest struct {
	String string
}
type nestedAComputedPropertiesForTest struct {
	PBool  *bool
	String string
}
type nestedBComputedPropertiesForTest struct {
	PI     *int64
	String string
}
type computedPropertiesTestModule struct {
	ModuleBase
	properties struct {
		String      string
		PString     *string
		Strings     []string
		Stringss    [][]string
		PStrings    []*string
		PI          *int64
		Ints        []int64
		Bool        bool
		PBool       *bool
		Bools       []bool
		Untouchable string `blueprint:"mutated"`
		Nested      nestedComputedPropertiesForTest
	}
	bizarroProperties struct {
		String  *string
		PString string
		Nested  *nestedAComputedPropertiesForTest
	}
	duplicateProperties struct {
		Strings []string
		Nested  nestedBComputedPropertiesForTest
	}
}

func (d computedPropertiesTestModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	ctx.Build(pctx, BuildParams{
		Rule:   Touch,
		Output: PathForModuleOut(ctx, "out"),
	})
}

func init() {
	RegisterConstantForStarlark("TEST_CONSTANT", stringPtr("test_constant"))
}

var derivationTestFixture = FixtureRegisterWithContext(func(ctx RegistrationContext) {
	ctx.RegisterModuleType("test", func() Module {
		module := &computedPropertiesTestModule{}
		module.nameProperties.Name = stringPtr("m")
		InitAndroidModule(module)
		module.AddProperties(&module.properties, &module.bizarroProperties, &module.duplicateProperties)
		return module
	})
	ctx.PreArchMutators(RegisterComputedPropertiesPreArchMutator)
})

func TestRegisterDerivedPropertiesPreArchMutator(t *testing.T) {
	successes := []struct {
		name       string
		bp         string
		assertions func(module *computedPropertiesTestModule)
	}{
		{"zeroes",
			`test { computed: "def compute(_):\n\treturn {}"}`,
			func(m *computedPropertiesTestModule) {
				if m == nil {
					t.Errorf("No failure expected")
				}
			},
		}, {"zeroes on nested",
			`test { computed: "def compute(_):\n\treturn { 'nested' : {}}"}`,
			func(m *computedPropertiesTestModule) {
				if m == nil {
					t.Errorf("No failure expected")
				}
			},
		}, {
			"computed *int",
			`test { computed: "def compute(_):\n\treturn {'PI': 20 - 30}"}`,
			func(m *computedPropertiesTestModule) {
				AssertIntEquals(t, "PI", -10, int(*m.properties.PI))
			},
		}, {
			"computed []int",
			`test { computed: "def compute(_):\n\treturn {'ints': [100, 200]}"}`,
			func(m *computedPropertiesTestModule) {
				AssertDeepEquals(t, "ints", []int64{100, 200}, m.properties.Ints)
			},
		}, {
			"computed bool",
			`test { computed: "def compute(_):\n\treturn {'bool': 1 == 1 and True}" }`,
			func(m *computedPropertiesTestModule) {
				AssertBoolEquals(t, "bool", true, m.properties.Bool)
			},
		}, {
			"computed *bool",
			`test { computed: "def compute(_):\n\treturn {'pBool': 1 < 2}"}`,
			func(m *computedPropertiesTestModule) {
				AssertBoolEquals(t, "pBool", true, *m.properties.PBool)
			},
		}, {
			"computed []bool",
			`test { computed: "def compute(_):\n\treturn {'bools': [True, False]}"}`,
			func(m *computedPropertiesTestModule) {
				AssertDeepEquals(t, "bools", []bool{true, false}, m.properties.Bools)
			},
		}, {
			"computed string",
			`test { computed: "def compute(_):\n\treturn {'string': 'abc'.upper()}" }`,
			func(m *computedPropertiesTestModule) {
				AssertStringEquals(t, "string", "ABC", m.properties.String)
				AssertStringEquals(t, "string", "ABC", *m.bizarroProperties.String)
			},
		}, {
			"computed *string",
			`test { computed: "def compute(_):\n\treturn {'pString': 'abc'.upper()}" }`,
			func(m *computedPropertiesTestModule) {
				AssertStringEquals(t, "pString", "ABC", *m.properties.PString)
				AssertStringEquals(t, "pString", "ABC", m.bizarroProperties.PString)
			},
		}, {
			"computed []string",
			`test { computed: "def compute(_):\n\treturn {'strings': ['a' + 'b']}" }`,
			func(m *computedPropertiesTestModule) {
				expected := []string{"ab"}
				if !reflect.DeepEqual(expected, m.properties.Strings) {
					t.Errorf("expected %v but was %v", expected, m.properties.Strings)
				}
			},
		}, {
			"computed [][]string",
			`test { computed: "def compute(_):\n\treturn {'stringss': [['a'], ['b', 'c']]}" }`,
			func(m *computedPropertiesTestModule) {
				expected := [][]string{{"a"}, {"b", "c"}}
				if !reflect.DeepEqual(expected, m.properties.Stringss) {
					t.Errorf("expected %v but was %v", expected, m.properties.Stringss)
				}
			},
		}, {
			"computed []*string",
			`test { computed: "def compute(_):\n\treturn {'pStrings': ['a', 'b']}" }`,
			func(m *computedPropertiesTestModule) {
				AssertIntEquals(t, "len(PStrings)", 2, len(m.properties.PStrings))
				AssertStringEquals(t, "PStrings[0]", "a", *m.properties.PStrings[0])
				AssertStringEquals(t, "PStrings[1]", "b", *m.properties.PStrings[1])
			},
		}, {
			"computed shared nested string and unshared nested properties",
			`test { computed: "def compute(_):\n\treturn {'nested': {'string': 'b', 'PI': 100, 'pBool': True}}" }`,
			func(m *computedPropertiesTestModule) {
				AssertStringEquals(t, "Nested.String", "b", m.properties.Nested.String)
				AssertStringEquals(t, "Nested.String", "b", m.bizarroProperties.Nested.String)
				AssertStringEquals(t, "Nested.String", "b", m.duplicateProperties.Nested.String)

				AssertIntEquals(t, "Nested.PI", 100, int(*m.duplicateProperties.Nested.PI))
				AssertBoolEquals(t, "Nested.PBool", true, *m.bizarroProperties.Nested.PBool)
			},
		}, {
			"disjoint nested properties",
			`test { computed: "def compute(_):\n\treturn {'nested': {'pBool': True, 'PI': 5}}" }`,
			func(m *computedPropertiesTestModule) {
				AssertBoolEquals(t, "Nested.PBool", true, *m.bizarroProperties.Nested.PBool)
				AssertIntEquals(t, "Nested.PInt", 5, int(*m.duplicateProperties.Nested.PI))
			},
		}, {
			"computed from ctx",
			`test { computed: "def compute(c):\n\treturn {'bool': not c.env.get('BOGUS'), 'string': c.constants.TEST_CONSTANT}" }`,
			func(m *computedPropertiesTestModule) {
				AssertBoolEquals(t, "from env", true, m.properties.Bool)
				AssertStringEquals(t, "from constants", "test_constant", m.properties.String)
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
		name            string
		bp              string
		failureMessages []string
	}{
		{
			"string collision",
			`test { string: "a", computed: "def compute(_):\n\treturn {'string': 'a'}"}`,
			//`test { string: "", computed: "{'string': 'a'}"}`,
			// ^ won't create a collision unfortunately because "" is a Zero value
			[]string{`property m\.string is already set`},
		}, {
			"bool collision",
			`test { bool: true, computed: "def compute(_):\n\treturn {'bool': True}"}`,
			//`test { bool: false, computed: "{'bool': True}"}`,
			// ^ won't create a collision unfortunately because false is a Zero value
			[]string{`property m\.bool is already set`},
		}, {
			"list collision",
			`test { strings: [], computed: "def compute(_):\n\treturn {'strings': []}"}`,
			[]string{`property m\.strings is already set`},
		}, {
			"pointer collision",
			`test { PI: 0, computed: "def compute(_):\n\treturn {'PI': 0}"}`,
			[]string{`property m\.PI is already set`},
		}, {
			"case sensitivity",
			`test { computed: "def compute(_):\n\treturn {'pI': 5, 'Bool': False}"}`,
			[]string{
				`property m\.pI does not exist - did you mean m\.PI`,
				`property m\.Bool does not exist - did you mean m\.bool`},
		}, {
			"blueprint mutated",
			`test { computed: "def compute(_):\n\treturn {'untouchable': 'b'}"}`,
			[]string{`property m\.untouchable is tagged blueprint:"mutated"`},
		}, {
			"non-function compute",
			`test { computed: "compute=55"}`,
			[]string{`expected \*starlark.Function but got starlark.Int`},
		}, {
			"missing compute function",
			`test { computed: "55"}`,
			[]string{"no such starlark function: compute"},
		}, {
			"starlark syntax",
			`test { computed: "def compute(_):\n\treturn}"}`,
			[]string{fmt.Sprintf("Android.bp-inline:2:%d: unexpected '}'", len("\treturn}"))},
		}, {
			"wrong return type",
			`test { computed: "def compute(_):\n\treturn {'string': 5}"}`,
			[]string{`property m\.string: starlark\.Int 5 cannot be cast to string`},
		}, {
			"mixed list type",
			`test { computed: "def compute(_):\n\treturn {'strings': ['a', 'b', 5]}"}`,
			[]string{`\[index:2\]: starlark\.Int 5 cannot be cast to string`},
		}, {
			"floats not supported",
			`test { computed: "def compute(_):\n\treturn {'PI': 5.0}"}`,
			[]string{`property m\.PI: starlark\.Float 5\.0 cannot be cast to int64`},
		}, {
			"unknown property",
			`test {computed: "def compute(_):\n\treturn {'what': 0}"}`,
			[]string{`m\.what does not exist`},
		}, {
			"unknown nested property",
			`test {computed: "def compute(_):\n\treturn {'nested': {'what': 0}}"}`,
			[]string{`m\.nested\.what does not exist`},
		}, {
			"unknown property and nested property",
			`test {computed: "def compute(_):\n\treturn {'what': True, 'nested': {'what': 0}}"}`,
			[]string{
				`m\.nested\.what does not exist`,
				`m\.what does not exist`,
			},
		}, {
			"unknown and known nested property",
			`test {computed: "def compute(_):\n\treturn {'nested': {'what': 0, 'String': 'a', 'who': 'joe'}}"}`,
			[]string{
				`m\.nested\.what does not exist`,
				`m\.nested\.who does not exist`,
			},
		}, {
			"return error",
			`test {computed: "def compute(_):\n\tfail('oops')"}`,
			[]string{"oops"},
		}, {
			"return non-dict",
			`test {computed: "def compute(_):\n\treturn 5"}`,
			[]string{"expected a dict but got starlark.Int"},
		}, {
			"reference non existent starlark script file",
			`test {computed: "file:blahblah"}`,
			[]string{"blahblah: file does not exist"},
		},
	}

	for _, test := range failures {
		t.Run(test.name, func(t *testing.T) {
			derivationTestFixture.
				ExtendWithErrorHandler(FixtureCustomErrorHandler(func(t *testing.T, result *TestResult) {
					t.Helper()
					for _, pattern := range test.failureMessages {
						if !FailIfNoMatchingErrors(t, pattern, result.Errs) {
							t.FailNow()
						}
					}
				})).
				RunTestWithBp(t, test.bp)
		})
	}
}
