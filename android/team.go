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

import "github.com/google/blueprint"

func init() {
	RegisterTeamBuildComponents(InitRegistrationContext)
}

func RegisterTeamBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("team", TeamFactory)
}

var PrepareForTestWithTeamBuildComponents = GroupFixturePreparers(
	FixtureRegisterWithContext(RegisterTeamBuildComponents),
)

type teamProperties struct {
	Trendy_team_id *string `json:"trendy_team_id"`
}

type teamModule struct {
	ModuleBase
	DefaultableModuleBase

	properties teamProperties
}

// We have two different providers here because languages (cc/java)
// need to set the provider in GenerateAndroidBuildActions.
// However, for java there is GenerateAndroidBuildActions for both *Library
// and *Test.  The Library method gets called for both tests and libraries, but
// a provider can only be set once and not mutated.
// The "test-only" is generally known at the library level, while the TopLevelTarget
// is known at Test level.
// It seems we need two different provider, not a larger struct to handle this situation.
// Client code should read both providers and OR them together and language code can
// optionally write with one or two providers.
type TestModuleInformation struct {
	TestOnly       bool
	TopLevelTarget bool
}

var TestOnlyProviderKey = blueprint.NewProvider[TestModuleInformation]()
var TestTargetProviderKey = blueprint.NewProvider[TestModuleInformation]()

// Real work is done for the module that depends on us.
// If needed, the team can serialize the config to json/proto file as well.
func (t *teamModule) GenerateAndroidBuildActions(ctx ModuleContext) {}

func (t *teamModule) TrendyTeamId(ctx ModuleContext) string {
	return *t.properties.Trendy_team_id
}

func TeamFactory() Module {
	module := &teamModule{}

	base := module.base()
	module.AddProperties(&base.nameProperties, &module.properties)

	InitAndroidModule(module)
	InitDefaultableModule(module)

	return module
}
