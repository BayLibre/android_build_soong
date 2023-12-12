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
	"fmt"
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
	intermediateCacheOutputPath android.OutputPath

	// Path to the text file containing all flags defined in the tree and their corresponding
	// boolean values that represents whether the flag is enabled or not.
	intermediateDumpOutputPath android.OutputPath

	// Similar to intermediateDumpOutputPath, but contains only the flags that are essential
	// to generate exportable stubs
	intermediateDumpExportableOutputPath android.OutputPath
}

func (aconfigDef *allAconfigDeclarationsSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// Find all of the aconfig_declarations modules
	var cacheFiles android.Paths
	ctx.VisitAllModules(func(module android.Module) {
		if !ctx.ModuleHasProvider(module, DeclarationsProviderKey) {
			return
		}
		decl := ctx.ModuleProvider(module, DeclarationsProviderKey).(DeclarationsProviderData)
		cacheFiles = append(cacheFiles, decl.IntermediateCacheOutputPath)
	})

	// Generate build action for aconfig
	aconfigDef.intermediateCacheOutputPath = android.PathForIntermediates(ctx, "all_aconfig_declarations.pb")
	ctx.Build(pctx, android.BuildParams{
		Rule:        AllDeclarationsRule,
		Inputs:      cacheFiles,
		Output:      aconfigDef.intermediateCacheOutputPath,
		Description: "all_aconfig_declarations_proto",
		Args: map[string]string{
			"format":      "protobuf",
			"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
		},
	})

	aconfigDef.intermediateDumpOutputPath = android.PathForIntermediates(ctx, "all_aconfig_declarations.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:        AllDeclarationsRule,
		Input:       aconfigDef.intermediateCacheOutputPath,
		Output:      aconfigDef.intermediateDumpOutputPath,
		Description: "all_aconfig_declarations_text",
		Args: map[string]string{
			"format":      "bool",
			"cache_files": "--cache " + aconfigDef.intermediateCacheOutputPath.String(),
		},
	})

	// TODO(b/315487153): Apply correct build rules when aconfig supports `--filter` argument
	aconfigDef.intermediateDumpExportableOutputPath = android.PathForIntermediates(ctx, "all_aconfig_declarations.exportable.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:        AllDeclarationsRule,
		Input:       aconfigDef.intermediateCacheOutputPath,
		Output:      aconfigDef.intermediateDumpExportableOutputPath,
		Description: "all_aconfig_declarations_text",
		Args: map[string]string{
			"format":      "bool",
			"cache_files": "--cache " + aconfigDef.intermediateCacheOutputPath.String(),
		},
	})

	ctx.Phony("all_aconfig_declarations",
		aconfigDef.intermediateCacheOutputPath,
		aconfigDef.intermediateDumpOutputPath,
		aconfigDef.intermediateDumpExportableOutputPath,
	)
}

func (aconfigDef *allAconfigDeclarationsSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.DistForGoal("droid",
		aconfigDef.intermediateCacheOutputPath,
		aconfigDef.intermediateDumpOutputPath,
		aconfigDef.intermediateDumpExportableOutputPath,
	)
}

var _ android.OutputFileProducer = (*allAconfigDeclarationsSingleton)(nil)

func (aconfigDef *allAconfigDeclarationsSingleton) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "", ".pb":
		return android.Paths{aconfigDef.intermediateCacheOutputPath}, nil
	case ".txt":
		return android.Paths{
			aconfigDef.intermediateDumpOutputPath,
			aconfigDef.intermediateDumpExportableOutputPath,
		}, nil
	case ".exportable.txt":
		return android.Paths{aconfigDef.intermediateDumpExportableOutputPath}, nil
	default:
		return nil, fmt.Errorf("unsupported module reference tag %q", tag)
	}
}
