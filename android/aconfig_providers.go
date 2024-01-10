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

package android

import (
	"fmt"

	"github.com/google/blueprint"
)

var (
	mergeAconfigFilesRule = pctx.AndroidStaticRule("mergeAconfigFilesRule",
		blueprint.RuleParams{
			Command:     `${aconfig} dump --dedup --format protobuf --out $out $flags`,
			CommandDeps: []string{"${aconfig}"},
		}, "flags")
	_ = pctx.HostBinToolVariable("aconfig", "aconfig")
)

// Provider published by aconfig_value_set
type AconfigDeclarationsProviderData struct {
	Package                     string
	Container                   string
	IntermediateCacheOutputPath WritablePath
	IntermediateDumpOutputPath  WritablePath
}

var AconfigDeclarationsProviderKey = blueprint.NewProvider[AconfigDeclarationsProviderData]()

type AconfigPropagatingDeclarationsInfo struct {
	AconfigFiles map[string]Paths
}

var AconfigPropagatingProvider = blueprint.NewProvider[AconfigPropagatingDeclarationsInfo]()

func init() {
	RegisterPropagatingProvider(AconfigPropagatingProvider, &AconfigPropagatingDeclarationsInfo{}, "")
}

func (*AconfigPropagatingDeclarationsInfo) Propagate(ctx ModuleContext) {
	var mergedAconfigFiles map[string]Paths = make(map[string]Paths)
	ctx.VisitDirectDepsIgnoreBlueprint(func(module Module) {
		if dep, _ := OtherModuleProvider(ctx, module, AconfigDeclarationsProviderKey); dep.IntermediateCacheOutputPath != nil {
			mergedAconfigFiles[dep.Container] = append(mergedAconfigFiles[dep.Container], dep.IntermediateCacheOutputPath)
		}
		// There are some module types (e.g., apex.apexBundle) which don't yet propagate through this Provider.
		// Add them here for now.
		if dep, _ := OtherModuleProvider(ctx, module, AconfigTransitiveDeclarationsInfoProvider); len(dep.AconfigFiles) > 0 {
			for container, v := range dep.AconfigFiles {
				mergedAconfigFiles[container] = append(mergedAconfigFiles[container], v...)
			}
		}
		if dep, _ := OtherModuleProvider(ctx, module, AconfigPropagatingProvider); len(dep.AconfigFiles) > 0 {
			for container, v := range dep.AconfigFiles {
				mergedAconfigFiles[container] = append(mergedAconfigFiles[container], v...)
			}
		}
	})
	amod := ctx.Module().base()
	var needProvider bool
	for container, aconfigFiles := range mergedAconfigFiles {
		mergedAconfigFiles[container] = mergeAconfigFiles(ctx, amod, container, aconfigFiles)
		needProvider = true
	}

	if needProvider {
		SetProvider(ctx, AconfigPropagatingProvider, AconfigPropagatingDeclarationsInfo{
			AconfigFiles: mergedAconfigFiles,
		})
	}
}

func (*AconfigPropagatingDeclarationsInfo) UpdateAndroidMkEntries(ctx SingletonContext, mod *Module, entries *[]AndroidMkEntries) {
	info, ok := SingletonModuleProvider(ctx, (*mod), AconfigPropagatingProvider)
	if !ok || len(info.AconfigFiles) == 0 {
		return
	}
	if len(*entries) == 0 {
		// TODO: if there are no entries, but we have aconfig dependencies, we probably need to
		// add one in this function.  For now, we can exit early.
		//fakeName := (*mod).String() + "-phony"
		//*entries = append(*entries, AndroidMkEntries{
		//	Class:      "FAKE",
		//	Include:    "$(BUILD_PHONY_PACKAGE)",
		//	OutputFile: OptionalPathForPath(PathForIntermediates(ctx, fakeName)),
		//})
		return
	}
	// All of the files in the module potentially depend on the flag values.
	toAdd := []AndroidMkExtraEntriesFunc{
		func(ctx AndroidMkExtraEntriesContext, entries *AndroidMkEntries) {
			setAconfigFileMkEntries((*mod).base(), entries, info.AconfigFiles)
		},
	}
	for idx, _ := range *entries {
		(*entries)[idx].ExtraEntries = append((*entries)[idx].ExtraEntries, toAdd...)
	}
}

// This is used to collect the aconfig declarations info on the transitive closure,
// the data is keyed on the container.
type AconfigTransitiveDeclarationsInfo struct {
	AconfigFiles map[string]Paths
}

var AconfigTransitiveDeclarationsInfoProvider = blueprint.NewProvider[AconfigTransitiveDeclarationsInfo]()

func CollectDependencyAconfigFiles(ctx ModuleContext, mergedAconfigFiles *map[string]Paths) {
	overrideHandling := true
	if overrideHandling {
		//return
	}
	if *mergedAconfigFiles == nil {
		*mergedAconfigFiles = make(map[string]Paths)
	}
	ctx.VisitDirectDepsBlueprint(func(module blueprint.Module) {
		// Walk our direct dependencies, ignoring blueprint Modules and disabled Android Modules.
		aModule, _ := module.(Module)
		if aModule == nil || !aModule.Enabled() {
			return
		}

		if dep, _ := OtherModuleProvider(ctx, module, AconfigDeclarationsProviderKey); dep.IntermediateCacheOutputPath != nil {
			(*mergedAconfigFiles)[dep.Container] = append((*mergedAconfigFiles)[dep.Container], dep.IntermediateCacheOutputPath)
			return
		}
		if dep, _ := OtherModuleProvider(ctx, module, AconfigTransitiveDeclarationsInfoProvider); len(dep.AconfigFiles) > 0 {
			for container, v := range dep.AconfigFiles {
				(*mergedAconfigFiles)[container] = append((*mergedAconfigFiles)[container], v...)
			}
		}
	})

	amod := ctx.Module().base()
	var needProvider bool
	for container, aconfigFiles := range *mergedAconfigFiles {
		(*mergedAconfigFiles)[container] = mergeAconfigFiles(ctx, amod, container, aconfigFiles)
		needProvider = true
	}

	if !overrideHandling && needProvider {
		SetProvider(ctx, AconfigTransitiveDeclarationsInfoProvider, AconfigTransitiveDeclarationsInfo{
			AconfigFiles: *mergedAconfigFiles,
		})
	}
}

func mergeAconfigFiles(ctx ModuleContext, mod *ModuleBase, container string, inputs Paths) Paths {
	inputs = LastUniquePaths(inputs)
	if len(inputs) == 1 {
		return Paths{inputs[0]}
	}

	var output ModuleOutPath
	if mod.savedAconfigMergedPb.path == "" {
		output = PathForModuleOut(ctx, container, "aconfig_merged.pb")

		ctx.Build(pctx, BuildParams{
			Rule:        mergeAconfigFilesRule,
			Description: "merge aconfig files",
			Inputs:      inputs,
			Output:      output,
			Args: map[string]string{
				"flags": JoinWithPrefix(inputs.Strings(), "--cache "),
			},
		})
		mod.savedAconfigMergedPb = output
	} else {
		output = mod.savedAconfigMergedPb
	}

	return Paths{output}
}

func SetAconfigFileMkEntries(m *ModuleBase, entries *AndroidMkEntries, aconfigFiles map[string]Paths) {
	setAconfigFileMkEntries(m, entries, aconfigFiles)
}
func setAconfigFileMkEntries(m *ModuleBase, entries *AndroidMkEntries, aconfigFiles map[string]Paths) {
	// TODO(b/311155208): The default container here should be system.
	container := ""

	if m.SocSpecific() {
		container = "vendor"
	} else if m.ProductSpecific() {
		container = "product"
	} else if m.SystemExtSpecific() {
		container = "system_ext"
	}

	var paths Paths
	paths = append(paths, aconfigFiles[container]...)
	if container != "" {
		if len(aconfigFiles[container]) == 0 && len(aconfigFiles[""]) > 0 {
			// TODO(b/308625757): Either we guessed the container wrong, or the flag is misdeclared.
			// For now, just include the system (aka "") container if we get here.
			fmt.Printf("LJ: container_mismatch(%v) container=%v files=%v\n", m, container, aconfigFiles)
		}
		paths = append(paths, aconfigFiles[""]...)
	}
	entries.AddPaths("LOCAL_ACONFIG_FILES", paths)
	m.SavedAconfigFiles = paths
}
