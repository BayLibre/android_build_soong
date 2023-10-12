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
	"strconv"

	"github.com/google/blueprint"
)

func init() {
	RegisterCodeMetadataBuildComponents(InitRegistrationContext)
}

// Registers the code metadata module type.
func RegisterCodeMetadataBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("code_metadata", CodeMetadataFactory)
}

func CodeMetadataFactory() Module {
	module := &CodeMetadataModule{}

	InitAndroidModule(module)
	InitDefaultableModule(module)
	module.AddProperties(&module.properties)

	return module
}

type CodeMetadataModule struct {
	ModuleBase
	DefaultableModuleBase
	BazelModuleBase

	// Properties for "code_metadata"
	properties struct {
		// Specifies the name of the code_config.
		Name string
		// Specifies the team ID.
		TeamId string
		// Specifies the list of modules that this code_metadata covers.
		Code []string

		// Source files for the modules covered by this code_metadata rule.
		CodeSrcs []Paths `blueprint:"mutated"`
	}
}

type codeDepTagType struct {
	blueprint.BaseDependencyTag
}

var codeDepTag = codeDepTagType{}

func (module *CodeMetadataModule) DepsMutator(ctx BottomUpMutatorContext) {
	// Validate Properties
	if len(module.properties.TeamId) == 0 {
		ctx.PropertyErrorf("TeamId", "Team Id not found in the code_metadata module. Hint: Maybe the teamId property hasn't been properly specified.")
	}
	if !isInt(module.properties.TeamId) {
		ctx.PropertyErrorf("TeamId", "Invalid value for Team ID. The Team ID must be an integer.")
	}
	if len(module.properties.Code) == 0 {
		ctx.PropertyErrorf("Code", "Targets to be attributed cannot be empty. Hint: Maybe the code property hasn't been properly specified.")
	}
	ctx.AddDependency(ctx.Module(), codeDepTag, module.properties.Code...)
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

type SrcsFileProviderData struct {
	SrcPaths Paths
}

var SrcsFileProviderKey = blueprint.NewProvider(SrcsFileProviderData{})

func (module *CodeMetadataModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	for _, m := range ctx.GetDirectDepsWithTag(codeDepTag) {
		if ctx.OtherModuleHasProvider(m, SrcsFileProviderKey) {
			module.properties.CodeSrcs = append(module.properties.CodeSrcs, ctx.OtherModuleProvider(m, SrcsFileProviderKey).(SrcsFileProviderData).SrcPaths)
		}
	}
}
