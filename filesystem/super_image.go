// Copyright (C) 2024 The Android Open Source Project
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

package filesystem

import (
	"fmt"

	"android/soong/android"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	android.RegisterModuleType("super_image", SuperImageFactory)
}

type superImage struct {
	android.ModuleBase

	properties     SuperImageProperties
	partitionProps SuperImagePartitionNameProperties

	installDir android.InstallPath
}

type SuperImageProperties struct {
	Size                     *string
	Metadata_device          *string
	AbUpdate                 *bool
	Retrofit                 *bool
	VirtualAb                *bool
	VirtualAbRetrofit        *bool
	BuildingSystemOtherImage bool
	Sparse                   bool
	Groups                   map[string]android.BoardSuperPartitionGroupProps
}

type SuperImagePartitionNameProperties struct {
	// Name of the System partition filesystem module
	System_partition_name *string
	// Name of the System_ext partition filesystem module
	System_ext_partition_name *string
	// Name of the System_dlkm partition filesystem module
	System_dlkm_partition_name *string
	// Name of the Product partition filesystem module
	Product_partition_name *string
	// Name of the Vendor partition filesystem module
	Vendor_partition_name *string
	// Name of the Vendor_dlkm partition filesystem module
	Vendor_dlkm_partition_name *string
	// Name of the Odm partition filesystem module
	Odm_partition_name *string
	// Name of the Odm_dlkm partition filesystem module
	Odm_dlkm_partition_name *string
}

func SuperImageFactory() android.Module {
	module := &superImage{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	return module
}

type superImageDepTagType struct {
	blueprint.BaseDependencyTag
}

var superImageDepTag superImageDepTagType

func (s *superImage) DepsMutator(ctx android.BottomUpMutatorContext) {
	addDependencyIfDefined := func(dep *string) {
		if dep != nil {
			ctx.AddDependency(ctx.Module(), superImageDepTag, proptools.String(dep))
		}
	}

	addDependencyIfDefined(s.partitionProps.System_partition_name)
	addDependencyIfDefined(s.partitionProps.System_ext_partition_name)
	addDependencyIfDefined(s.partitionProps.System_dlkm_partition_name)
	addDependencyIfDefined(s.partitionProps.Product_partition_name)
	addDependencyIfDefined(s.partitionProps.Vendor_partition_name)
	addDependencyIfDefined(s.partitionProps.Vendor_dlkm_partition_name)
	addDependencyIfDefined(s.partitionProps.Odm_partition_name)
	addDependencyIfDefined(s.partitionProps.Odm_dlkm_partition_name)
}

func (s *superImage) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	s.buildSuperImage(ctx)
}

func (s *superImage) installFileName() string {
	return s.BaseModuleName() + ".img"
}

func (s *superImage) buildSuperImage(ctx android.ModuleContext) {
	builder := android.NewRuleBuilder(pctx, ctx)
	cmd := builder.Command().BuiltTool("lpmake")
	cmd.FlagWithArg("--metadata-size", "65536")
	cmd.FlagWithArg("--super-name", proptools.String(s.properties.Metadata_device))

	abUpdate := proptools.Bool(s.properties.AbUpdate)
	retrofit := proptools.Bool(s.properties.Retrofit)
	virtualAb := proptools.Bool(s.properties.VirtualAb)
	virtualAbRetrofit := proptools.Bool(s.properties.VirtualAbRetrofit)

	if abUpdate && !retrofit {
		cmd.FlagWithArg("--metadata-slots=", "3")
	} else {
		cmd.FlagWithArg("--metadata-slots=", "2")
	}

	if abUpdate && retrofit {
		cmd.Flag("--auto-slot-suffixing")
	}
	if virtualAb && !virtualAbRetrofit {
		cmd.Flag("--virtual-ab")
	}

	cmd.FlagWithArg("--device", "super:"+proptools.String(s.properties.Size))

	appendSuffix := abUpdate && !retrofit
	for _, group := range android.SortedKeys(s.properties.Groups) {
		if appendSuffix {
			cmd.FlagWithArg("--group", group+"_a:"+s.properties.Groups[group].GroupSize)
			cmd.FlagWithArg("--group", group+"_b:"+s.properties.Groups[group].GroupSize)
		} else {
			cmd.FlagWithArg("--group", group+":"+s.properties.Groups[group].GroupSize)
		}

		s.updateCommand(ctx, cmd, appendSuffix, group)
	}

	if s.properties.Sparse {
		cmd.Flag("--sparse")
	}
	output := android.PathForModuleOut(ctx, s.installFileName())
	cmd.Output(output)
	builder.Build("buildSuperImage", fmt.Sprintf("Generating super image for %s", s.BaseModuleName()))
}

func (s *superImage) updateCommand(ctx android.ModuleContext, cmd *android.RuleBuilderCommand, appendSuffix bool, group string) bool {
	var hasImage bool
	partitionToImage := make(map[string]string)

	// first pass: build map from partition name to its iamge install path
	for _, partition := range s.properties.Groups[group].PartitionList {
		var partitionNameNeeded *string
		var imagePath string
		switch partition {
		case "system":
			partitionNameNeeded = s.partitionProps.System_partition_name
		case "system_dlkm":
			partitionNameNeeded = s.partitionProps.System_dlkm_partition_name
		case "system_ext":
			partitionNameNeeded = s.partitionProps.System_ext_partition_name
		case "product":
			partitionNameNeeded = s.partitionProps.Product_partition_name
		case "vendor":
			partitionNameNeeded = s.partitionProps.Vendor_partition_name
		case "vendor_dlkm":
			partitionNameNeeded = s.partitionProps.Vendor_dlkm_partition_name
		case "odm":
			partitionNameNeeded = s.partitionProps.Odm_partition_name
		case "odm_dlkm":
			partitionNameNeeded = s.partitionProps.Odm_dlkm_partition_name
		default:
			ctx.ModuleErrorf("current partition %s not in SuperImagePartitionNameProperties", partition)
		}

		if proptools.String(partitionNameNeeded) != "" {
			ctx.VisitDirectDeps(func(m android.Module) {
				if f, ok := m.(*filesystem); ok {
					if f.Name() == *partitionNameNeeded {
						// imagePath = f installpath
						imagePath = "aaaaa"
						return
					}
				}
			})
		}
		partitionToImage[partition] = imagePath
	}

	// second pass: update build command
	for _, partition := range s.properties.Groups[group].PartitionList {
		image := partitionToImage[partition]
		if image != "" {
			hasImage = true
		}

		if !appendSuffix {
			updateCommandForImage(cmd, group, partition, image)
			continue
		}
		updateCommandForImage(cmd, group+"_a", partition+"_a", image)
		if partition == "system" && s.properties.BuildingSystemOtherImage {
			otherImage := partitionToImage["system_other"]
			updateCommandForImage(cmd, group+"_b", partition+"_b", otherImage)
		}
	}

	return hasImage
}

func updateCommandForImage(cmd *android.RuleBuilderCommand, group, partition, image string) {
	// how to get image size
}
