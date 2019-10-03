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

package apex

import (
	"android/soong/android"
	"android/soong/java"
)

func init() {
	android.RegisterModuleType("prebuilt_boot", PrebuiltBootFactory)
}

type PrebuiltBoot struct {
	android.ModuleBase

	outputFilePath android.OutputPath
}

func (p *PrebuiltBoot) DepsMutator(ctx android.BottomUpMutatorContext) {
	ctx.AddDependency(ctx.Module(), nil, "dex_bootjars_global")
}

func (p *PrebuiltBoot) OutputFile() android.OutputPath {
	return p.outputFilePath
}

func (p *PrebuiltBoot) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// find unique DexpreoptBootJars module
	var dexpreopt *java.DexpreoptBootJars
	ctx.VisitDirectDeps(func(dep android.Module) {
		if j, ok := dep.(*java.DexpreoptBootJars); ok {
			if dexpreopt != nil {
				ctx.ModuleErrorf("multiple DexpreoptBootJars modules found: %s, %s",
					dexpreopt.Name(), j.Name())
			}
			dexpreopt = j
		}
	})
	if dexpreopt == nil {
		ctx.ModuleErrorf("no DexpreoptBootJars module found")
	}

	arch := ctx.Target().Arch.ArchType
	sourceFilePath := dexpreopt.Images[arch]
	p.outputFilePath = android.PathForModuleOut(ctx, "boot-"+arch.String()+".art").OutputPath

	ctx.Build(pctx, android.BuildParams{
		Rule:   android.Cp,
		Output: p.outputFilePath,
		Input:  sourceFilePath,
	})
}

func PrebuiltBootFactory() android.Module {
	module := &PrebuiltBoot{}
	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibBoth)
	return module
}
