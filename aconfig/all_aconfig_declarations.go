// Copyright 2023 Google Inc. All rights reserved.
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

package aconfig

import (
	"android/soong/android"
)

// A singleton module that collects all of the aconfig flags declared in the
// tree into a single combined file for export to the external flag setting
// server (inside Google it's Gantry).
//
// Note that this is ALL aconfig_declarations modules present in the tree, not just
// ones that are relevant to the product currently being built, so that that infra
// doesn't need to pull from multiple builds and merge them.
func AllAconfigDeclarationsFactory() android.Singleton {
	return &allAconfigDeclarationsSingleton{}
}

type allAconfigDeclarationsSingleton struct {
	intermediateProtoPath android.OutputPath
	intermediateTextPath  android.OutputPath
}

func (aconfigDef *allAconfigDeclarationsSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// Find all of the aconfig_declarations modules
	var cacheFiles android.Paths
	ctx.VisitAllModules(func(module android.Module) {
		if !ctx.ModuleHasProvider(module, DeclarationsProviderKey) {
			return
		}
		decl := ctx.ModuleProvider(module, DeclarationsProviderKey).(DeclarationsProviderData)
		cacheFiles = append(cacheFiles, decl.IntermediateProtoPath)
	})

	// Generate build action for aconfig
	aconfigDef.intermediateProtoPath = android.PathForIntermediates(ctx, "all_aconfig_declarations.pb")
	ctx.Build(pctx, android.BuildParams{
		Rule:        AllDeclarationsRule,
		Inputs:      cacheFiles,
		Output:      aconfigDef.intermediateProtoPath,
		Description: "all_aconfig_declarations_proto",
		Args: map[string]string{
			"format":      "protobuf",
			"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
		},
	})

	aconfigDef.intermediateTextPath = android.PathForIntermediates(ctx, "all_aconfig_declarations.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:        AllDeclarationsRule,
		Inputs:      cacheFiles,
		Output:      aconfigDef.intermediateTextPath,
		Description: "all_aconfig_declarations_text",
		Args: map[string]string{
			"format":      "map",
			"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
		},
	})

	ctx.Phony("all_aconfig_declarations", aconfigDef.intermediateProtoPath, aconfigDef.intermediateTextPath)
}

func (aconfigDef *allAconfigDeclarationsSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.DistForGoal("droid", aconfigDef.intermediateProtoPath, aconfigDef.intermediateTextPath)
}
