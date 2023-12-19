// Copyright 2024 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package android

import (
	"testing"
)

type dummyTypeForTests struct {
	ModuleBase
}

func dummyTypeFactory() Module {
	module := &dummyTypeForTests{}
	InitAndroidModule(module)
	return module
}

func (*dummyTypeForTests) GenerateAndroidBuildActions(_ ModuleContext) {}

// func (*dummyTypeForTests) GenerateAndroidBuildActions(_ android.ModuleContext) {}

func TestTeam(t *testing.T) {
	t.Parallel()
	ctx := GroupFixturePreparers(
		PrepareForTestWithTeamBuildComponents,
		FixtureRegisterWithContext(func(ctx RegistrationContext) {
			ctx.RegisterModuleType("dummy", dummyTypeFactory)
		}),
	).RunTestWithBp(t, `
		dummy {
			name: "main_test",
			team: "someteam",
		}
		team {
			name: "someteam",
			trendy_team_id: "cool_team",
		}

		team {
			name: "team2",
			trendy_team_id: "22222",
		}

		dummy {
			name: "tool",
			team: "team2",
		}
	`)

	// Assert the rule from GenerateAndroidBuildActions exists.
	expectedDescription := "raw intermediateOwnerData.textproto"
	ctx.ModuleForTests("main_test", "").Description(expectedDescription)
}

func TestMissingTeamFails(t *testing.T) {
	t.Parallel()
	GroupFixturePreparers(
		PrepareForTestWithTeamBuildComponents,
		FixtureRegisterWithContext(func(ctx RegistrationContext) {
			ctx.RegisterModuleType("dummy", dummyTypeFactory)
		}),
	).
		ExtendWithErrorHandler(FixtureExpectsAtLeastOneErrorMatchingPattern("depends on undefined module \"smeagol")).
		RunTestWithBp(t, `
		dummy {
			name: "you_cannot_pass",
			team: "smeagol",
		}
	`)
}

func TestPackageDefaultTeam(t *testing.T) {
	t.Parallel()
	ctx := GroupFixturePreparers(
		PrepareForTestWithTeamBuildComponents,
		PrepareForTestWithPackageModule,
		FixtureRegisterWithContext(func(ctx RegistrationContext) {
			ctx.RegisterModuleType("dummy", dummyTypeFactory)
		}),
	).RunTestWithBp(t, `
                package {
                    default_team: "team2"
                }
		dummy {
			name: "main_test2",
		}
		team {
			name: "team2",
			trendy_team_id: "22222",
		}

	`)

	// Assert the rule from GenerateAndroidBuildActions exists.
	expectedDescription := "raw intermediateOwnerData.textproto"
	ctx.ModuleForTests("main_test2", "").Description(expectedDescription)
}
