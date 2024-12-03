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
	"slices"
	"strconv"
	"strings"

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
	MetadataDevice           *string
	BlockDevices             *string
	AbUpdate                 *bool
	Retrofit                 *bool
	VirtualAb                *bool
	VirtualAbRetrofit        *bool
	BuildingSystemOtherImage *bool
	Sparse                   *bool
	PartitionList            []string
	PartitionGroups          []string
	PartitionGroupsInfo      map[string]android.BoardSuperPartitionGroupProps
	UseDynamicPartitions     *bool
	LpMakeDir                *string
}

type SuperImagePartitionNameProperties struct {
	// Name of the System partition filesystem module
	System_partition_name *string
	// Name of the System_ext partition filesystem module
	System_ext_partition_name *string
	// Name of the System_dlkm partition filesystem module
	System_dlkm_partition_name *string
	// Name of the System_other partition filesystem module
	System_other_partition_name *string
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
	miscInfo := s.buildMiscInfo(ctx)
	builder := android.NewRuleBuilder(pctx, ctx)
	output := android.PathForModuleOut(ctx, s.installFileName())
	builder.Command().Textf("mkdir -p %s", ctx.Config().OutDir()).
		Textf("PATH=%s:\\$PATH", s.properties.LpMakeDir).
		BuiltTool("build_super_image").
		Text("-v").
		Input(miscInfo).
		Output(output)
	builder.Build("build_super_image", fmt.Sprintf("Creating super image %s", s.BaseModuleName()))
}

func (s *superImage) installFileName() string {
	return s.BaseModuleName() + ".img"
}

func (s *superImage) buildMiscInfo(ctx android.ModuleContext) android.Path {
	var miscInfoString strings.Builder
	addStr := func(name string, value string) {
		miscInfoString.WriteString(name)
		miscInfoString.WriteRune('=')
		miscInfoString.WriteString(value)
		miscInfoString.WriteRune('\n')
	}

	addStr("use_dynamic_partitions", strconv.FormatBool(proptools.Bool(s.properties.UseDynamicPartitions)))
	addStr("dynamic_partition_retrofit", strconv.FormatBool(proptools.Bool(s.properties.Retrofit)))
	addStr("lpmake", "lpmake")
	addStr("super_metadata_device", proptools.String(s.properties.MetadataDevice))
	addStr("super_block_devices", proptools.String(s.properties.BlockDevices))
	addStr("super_super_device_size", proptools.String(s.properties.Size))
	addStr("dynamic_partition_list", strings.Join(s.properties.PartitionList, " "))
	addStr("super_partition_groups", strings.Join(s.properties.PartitionGroups, " "))
	for _, group := range android.SortedKeys(s.properties.PartitionGroupsInfo) {
		addStr("super_"+group+"_group_size", s.properties.PartitionGroupsInfo[group].GroupSize)
		addStr("super_"+group+"_partition_list", strings.Join(s.properties.PartitionGroupsInfo[group].PartitionList, " "))
	}
	addStr("virtual_ab", strconv.FormatBool(proptools.Bool(s.properties.VirtualAb)))
	addStr("virtual_ab_retrofit", strconv.FormatBool(proptools.Bool(s.properties.VirtualAbRetrofit)))

	partitionToImagePath := make(map[string]string)
	var systemOtherPartitionNameNeeded string
	partitionNamesNeeded := []string{}
	addToPartitionNamesNeededIfPossible := func(s *string) {
		if proptools.String(s) != "" {
			partitionNamesNeeded = append(partitionNamesNeeded, *s)
		}
	}

	// Build partitionToImagePath, because system partition may need system_other
	// partition image path
	for _, p := range s.properties.PartitionList {
		switch p {
		case "system":
			addToPartitionNamesNeededIfPossible(s.partitionProps.System_partition_name)
			if proptools.Bool(s.properties.BuildingSystemOtherImage) {
				systemOtherPartitionNameNeeded = proptools.String(s.partitionProps.System_other_partition_name)
			}
		case "system_dlkm":
			addToPartitionNamesNeededIfPossible(s.partitionProps.System_dlkm_partition_name)
		case "system_ext":
			addToPartitionNamesNeededIfPossible(s.partitionProps.System_ext_partition_name)
		case "product":
			addToPartitionNamesNeededIfPossible(s.partitionProps.Product_partition_name)
		case "vendor":
			addToPartitionNamesNeededIfPossible(s.partitionProps.Vendor_partition_name)
		case "vendor_dlkm":
			addToPartitionNamesNeededIfPossible(s.partitionProps.Vendor_dlkm_partition_name)
		case "odm":
			addToPartitionNamesNeededIfPossible(s.partitionProps.Odm_partition_name)
		case "odm_dlkm":
			addToPartitionNamesNeededIfPossible(s.partitionProps.Odm_dlkm_partition_name)
		default:
			ctx.ModuleErrorf("current partition %s not a super image supported partition", p)
		}

		ctx.VisitDirectDeps(func(m android.Module) {
			if slices.Contains(partitionNamesNeeded, m.Name()) {
				if output, ok := android.ModuleProvider(ctx, android.OutputFilesProvider); ok {
					partitionToImagePath[p] = output.DefaultOutputFiles[0].String()
				}
			} else if systemOtherPartitionNameNeeded != "" && m.Name() == systemOtherPartitionNameNeeded {
				if output, ok := android.ModuleProvider(ctx, android.OutputFilesProvider); ok {
					partitionToImagePath["system_other"] = output.DefaultOutputFiles[0].String()
				}
			}
		})

		for _, p := range android.SortedKeys(partitionToImagePath) {
			addStr(p+"_image", partitionToImagePath[p])
		}

	}

	miscInfo := android.PathForModuleOut(ctx, "misc_info.txt")
	android.WriteFileRuleVerbatim(ctx, miscInfo, miscInfoString.String())
	return miscInfo
}
