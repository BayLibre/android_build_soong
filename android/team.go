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

import "strconv"

func init() {
	RegisterTeamBuildComponents(InitRegistrationContext)
}

// Register the license_kind module type.
func RegisterTeamBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("team", TeamFactory)
}

type teamProperties struct {
	Trendy_team_id *int64 `json:"trendy_team_id"`
}

var _ Bazelable = &teamModule{}

type teamModule struct {
	ModuleBase
	DefaultableModuleBase
	BazelModuleBase

	properties teamProperties

	output OutputPath
}

func (t *teamModule) DepsMutator(ctx BottomUpMutatorContext) {
	// Nothing to do.
}

func (t *teamModule) AndroidMkEntries() []AndroidMkEntries {
	return []AndroidMkEntries{{
		Class:      "FAKE",
		OutputFile: OptionalPathForPath(t.output),
		Include:    "$(BUILD_PHONY_PACKAGE)",
		ExtraEntries: []AndroidMkExtraEntriesFunc{
			func(ctx AndroidMkExtraEntriesContext, entries *AndroidMkEntries) {
				entries.SetString("LOCAL_TRENDY_TEAM_ID", strconv.FormatInt(*t.properties.Trendy_team_id, 10))
			},
		},
	}}
}

func (t *teamModule) ConvertWithBp2build(ctx Bp2buildMutatorContext) {
}

func (t *teamModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	// Nothing to do.
	// Maybe write a proto later.
	outputFile := PathForModuleOut(ctx, "team_details.txt").OutputPath
	t.output = outputFile
	/// TODO(ron): WriteFile
}

func TeamFactory() Module {
	module := &teamModule{}

	base := module.base()
	module.AddProperties(&base.nameProperties, &module.properties, &base.commonProperties.BazelConversionStatus)

	InitAndroidModule(module)
	InitDefaultableModule(module)
	InitBazelModule(module)

	return module
}
