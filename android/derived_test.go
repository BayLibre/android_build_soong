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
		Nested      struct{ String string }
	}
}

func (d derivationsTestModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	ctx.Build(pctx, BuildParams{
		Rule:   Touch,
		Output: PathForModuleOut(ctx, "out"),
	})
}

func TestRegisterDerivedPropertiesPreArchMutator(t *testing.T) {
	var derivationfixture = FixtureRegisterWithContext(func(ctx RegistrationContext) {
		ctx.RegisterModuleType("test", func() Module {
			module := &derivationsTestModule{}
			module.nameProperties.Name = stringPtr("testModule")
			InitAndroidModule(module)
			module.AddProperties(&module.properties)
			return module
		})
		ctx.PreArchMutators(RegisterDerivedPropertiesPreArchMutator)
	})

	successes := []struct {
		name       string
		bp         string
		assertions func(module *derivationsTestModule)
	}{
		{
			"derived *int",
			`test { derived: "{'ptrInt': 20 - 30}"}`,
			func(m *derivationsTestModule) {
				AssertIntEquals(t, "ptrInt", -10, int(*m.properties.PtrInt))
			},
		}, {
			"derived []int",
			`test { derived: "{'ints': [100, 200]}"}`,
			func(m *derivationsTestModule) {
				AssertDeepEquals(t, "ints", []int64{100, 200}, m.properties.Ints)
			},
		}, {
			"derived bool",
			`test { derived: "{'bool': 1 == 1 and True}" }`,
			func(m *derivationsTestModule) {
				AssertBoolEquals(t, "bool", true, m.properties.Bool)
			},
		}, {
			"derived *bool",
			`test { derived: "{'ptrBool': 1 < 2}"}`,
			func(m *derivationsTestModule) {
				AssertBoolEquals(t, "ptrBool", true, *m.properties.PtrBool)
			},
		}, {
			"derived []bool",
			`test { derived: "{'bools': [True, False]}"}`,
			func(m *derivationsTestModule) {
				AssertDeepEquals(t, "bools", []bool{true, false}, m.properties.Bools)
			},
		}, {
			"derived string",
			`test { derived: "{'string': 'abc'.upper()}" }`,
			func(m *derivationsTestModule) {
				AssertStringEquals(t, "string", "ABC", m.properties.String)
			},
		}, {
			"derived *string",
			`test { derived: "{'ptrString': 'abc'.upper()}" }`,
			func(m *derivationsTestModule) {
				AssertStringEquals(t, "ptrString", "ABC", *m.properties.PtrString)
			},
		}, {
			"derived []string",
			`test { derived: "{'strings': ['a' + 'b']}" }`,
			func(m *derivationsTestModule) {
				expected := []string{"ab"}
				if !reflect.DeepEqual(expected, m.properties.Strings) {
					t.Errorf("expected %v but was %v", expected, m.properties.Strings)
				}
			},
		}, {
			"derived [][]string",
			`test { derived: "{'stringss': [['a'], ['b', 'c']]}" }`,
			func(m *derivationsTestModule) {
				expected := [][]string{{"a"}, {"b", "c"}}
				if !reflect.DeepEqual(expected, m.properties.Stringss) {
					t.Errorf("expected %v but was %v", expected, m.properties.Stringss)
				}
			},
		}, {
			"derived []*string",
			`test { derived: "{'ptrStrings': ['a', 'b']}" }`,
			func(m *derivationsTestModule) {
				AssertIntEquals(t, "len(PtrStrings)", 2, len(m.properties.PtrStrings))
				AssertStringEquals(t, "PtrStrings[0]", "a", *m.properties.PtrStrings[0])
				AssertStringEquals(t, "PtrStrings[1]", "b", *m.properties.PtrStrings[1])
			},
		}, {
			"derived nested string",
			`test { derived: "{'nested': {'string': 'b'}}" }`,
			func(m *derivationsTestModule) {
				AssertStringEquals(t, "Nested.String", "b", m.properties.Nested.String)
			},
		},
	}

	for _, test := range successes {
		t.Run(test.name, func(t *testing.T) {
			result := derivationfixture.RunTestWithBp(t, test.bp)
			module := result.Module("testModule", "").(*derivationsTestModule)
			test.assertions(module)
		})
	}

	failures := []struct {
		name           string
		bp             string
		failureMessage string
	}{
		{
			"plain string and derived string collide",
			`test { string: "a", derived: "{'string': 'b'}"}`,
			`already set "String" = "a"`,
		}, {
			"blueprint mutated",
			`test { derived: "{'untouchable': 'b'}"}`,
			`"Untouchable" is marked blueprint:"mutated"`,
		},
	}

	for _, test := range failures {
		derivationfixture.
			ExtendWithErrorHandler(FixtureExpectsAtLeastOneErrorMatchingPattern(test.failureMessage)).
			RunTestWithBp(t, test.bp)
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
