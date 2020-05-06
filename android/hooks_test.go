// Copyright 2020 Google Inc. All rights reserved.
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
	"strings"
	"testing"

	"github.com/google/blueprint/proptools"
)

func testHooks(bp string) (*TestContext, []error) {
	// Create a test per config to allow for test specific config, e.g. test rules.
	config := TestConfig(buildDir, nil, bp, nil)

	ctx := NewTestContext()
	ctx.RegisterModuleType("mock_wrapper", newMockWrapperModule)
	ctx.Register(config)

	_, errs := ctx.ParseBlueprintsFiles("Android.bp")
	if len(errs) > 0 {
		return ctx, errs
	}

	_, errs = ctx.PrepareBuildActions(config)
	return ctx, errs
}

func TestFixProperties(t *testing.T) {
	ctx, errs := testHooks(`
mock_wrapper {
	name: "mock",
}
`)

	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%s", err.Error())
		}
		return
	}

	wrapperModule := ctx.ModuleForTests("mock", "").Module().(*mockWrapperModule)
	fixedString := proptools.String(wrapperModule.properties.Fixed_string)
	if fixedString != "fixed" {
		t.Errorf(`expected fixed_string to be "fixed" but is %q`, fixedString)
	}

	clearedBool := wrapperModule.properties.Cleared_bool
	if clearedBool != nil {
		t.Errorf(`expected cleared_bool to be nil" but is %v`, clearedBool)
	}
}

func TestFixProperties_Invalid_FixedString(t *testing.T) {
	_, errs := testHooks(`
mock_wrapper {
	name: "mock",
	fixed_string: "wrong",
}
`)

	if len(errs) != 0 {
		for _, err := range errs {
			if !strings.HasSuffix(err.Error(), ` module "mock": fixed_string: is fixed to "fixed" and cannot be set to "wrong"`) {
				t.Errorf("%s", err.Error())
			}
		}
		return
	}
}

func TestFixProperties_Invalid_FixedString_Redundant(t *testing.T) {
	_, errs := testHooks(`
mock_wrapper {
	name: "mock",
	fixed_string: "fixed",
}
`)

	if len(errs) != 0 {
		for _, err := range errs {
			if !strings.HasSuffix(err.Error(), ` module "mock": fixed_string: is fixed to "fixed" so setting it to the same value is redundant`) {
				t.Errorf("%s", err.Error())
			}
		}
		return
	}
}

func TestFixProperties_Invalid_ClearedBool(t *testing.T) {
	_, errs := testHooks(`
mock_wrapper {
	name: "mock",
	cleared_bool: false,
}
`)

	if len(errs) != 0 {
		for _, err := range errs {
			if !strings.HasSuffix(err.Error(), ` module "mock": cleared_bool: is fixed to nil and cannot be set to false`) {
				t.Errorf("%s", err.Error())
			}
		}
		return
	}
}

type mockWrapperProperties struct {
	Fixed_string *string
	Cleared_bool *bool
}

type mockWrapperModule struct {
	ModuleBase
	properties mockWrapperProperties
}

func newMockWrapperModule() Module {
	m := &mockWrapperModule{}
	m.AddProperties(&m.properties)
	InitAndroidModule(m)

	AddLoadHook(m, func(ctx LoadHookContext) {
		ctx.FixProperties(&mockWrapperProperties{
			Fixed_string: proptools.StringPtr("fixed"),
		})
	})
	return m
}

func (p *mockWrapperModule) GenerateAndroidBuildActions(ModuleContext) {
}
