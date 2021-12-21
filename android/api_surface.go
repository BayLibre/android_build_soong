// Copyright 2021 Google Inc. All rights reserved.
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
	"github.com/google/blueprint"
)

func init() {
	RegisterApiSurfaceBuildComponents(InitRegistrationContext)
}

var PrepareForTestWithApiSurface = FixtureRegisterWithContext(RegisterApiSurfaceBuildComponents)

func RegisterApiSurfaceBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("api_surface", ApiSurfaceFactory)
}

type ApiSurface struct {
	ModuleBase
	properties apiSurfaceProperties
}

type apiSurfaceProperties struct {
	Contributions []string
}

func ApiSurfaceFactory() Module {
	module := &ApiSurface{}
	module.AddProperties(&module.properties)
	InitAndroidModule(module)
	return module
}

func (surface *ApiSurface) DepsMutator(ctx BottomUpMutatorContext) {
	if surface.properties.Contributions != nil {
		ctx.AddVariationDependencies(nil, nil, surface.properties.Contributions...)
	}

}
func (surface *ApiSurface) GenerateAndroidBuildActions(ctx ModuleContext) {
	var contributionFiles Paths
	ctx.WalkDeps(func(child, parent Module) bool {
		if contribution, ok := child.(ApiContribution); ok {
			copied := contribution.CopyFiles(ctx)
			contributionFiles = append(contributionFiles, copied...)
			generatedBuildFiles := contribution.GenerateBuildFiles(ctx)
			contributionFiles = append(contributionFiles, generatedBuildFiles...)
			return false // no transitive dependencies
		}
		return false
	})

	// phony target
	ctx.Build(pctx, BuildParams{
		Rule:   blueprint.Phony,
		Output: PathForPhony(ctx, ctx.ModuleName()),
		Inputs: contributionFiles,
	})
}

type ApiContribution interface {
	// copy files necessaryt to construct an API surface
	// For C, it will be map.txt and .h files
	// For Java, it will be api.txt
	CopyFiles(ctx ModuleContext) Paths // output paths

	// Generate Android.bp in out/ to use the exported .txt files
	GenerateBuildFiles(ctx ModuleContext) Paths //output paths
}
