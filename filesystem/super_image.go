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
	// the size of the super partition
	Size *string
	// the block device where metadata for dynamic partitions is stored
	MetadataDevice *string
	// the super partition block device list
	BlockDevices *string
	// whether A/B updater is used
	AbUpdate *bool
	// whether dynamic partitions is enabled on devices that were launched without this support
	Retrofit *bool
	// whether virtual A/B seamless update is enabled
	VirtualAb *bool
	// whether retrofitting virtual A/B seamless update is enabled
	VirtualAbRetrofit *bool
	// whether system_other image should be built
	BuildingSystemOtherImage *bool
	// whether the output is a sparse image
	Sparse *bool
	// the partitions included in the super partition
	PartitionList []string
	// information about how partitions within the super partition are grouped together
	PartitionGroups []PartitionGroupsInfo
	// whether dynamic partitions is used
	UseDynamicPartitions *bool
	// the dir of lpmake tool
	LpMakeDir *string
}

type PartitionGroupsInfo struct {
	Name          string
	GroupSize     string
	PartitionList []string
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
	module.AddProperties(&module.properties, &module.partitionProps)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
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
	addDependencyIfDefined(s.partitionProps.System_other_partition_name)
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
		Textf("PATH=%s:\\$PATH", proptools.String(s.properties.LpMakeDir)).
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

	var groups []string
	for _, groupInfo := range s.properties.PartitionGroups {
		groups = append(groups, groupInfo.Name)
		addStr("super_"+groupInfo.Name+"_group_size", groupInfo.GroupSize)
		addStr("super_"+groupInfo.Name+"_partition_list", strings.Join(groupInfo.PartitionList, " "))
	}
	addStr("super_partition_groups", strings.Join(groups, " "))

	addStr("virtual_ab", strconv.FormatBool(proptools.Bool(s.properties.VirtualAb)))
	addStr("virtual_ab_retrofit", strconv.FormatBool(proptools.Bool(s.properties.VirtualAbRetrofit)))

	partitionToImagePath := make(map[string]string)
	nameToPartition := make(map[string]string)
	var systemOtherPartitionNameNeeded string
	addEntryToPartitionToName := func(p string, s *string) {
		if proptools.String(s) != "" {
			nameToPartition[*s] = p
		}
	}

	// Build partitionToImagePath, because system partition may need system_other
	// partition image path
	for _, p := range s.properties.PartitionList {
		if _, ok := nameToPartition[p]; ok {
			continue
		}
		switch p {
		case "system":
			addEntryToPartitionToName(p, s.partitionProps.System_partition_name)
			if proptools.Bool(s.properties.BuildingSystemOtherImage) {
				systemOtherPartitionNameNeeded = proptools.String(s.partitionProps.System_other_partition_name)
			}
		case "system_dlkm":
			addEntryToPartitionToName(p, s.partitionProps.System_dlkm_partition_name)
		case "system_ext":
			addEntryToPartitionToName(p, s.partitionProps.System_ext_partition_name)
		case "product":
			addEntryToPartitionToName(p, s.partitionProps.Product_partition_name)
		case "vendor":
			addEntryToPartitionToName(p, s.partitionProps.Vendor_partition_name)
		case "vendor_dlkm":
			addEntryToPartitionToName(p, s.partitionProps.Vendor_dlkm_partition_name)
		case "odm":
			addEntryToPartitionToName(p, s.partitionProps.Odm_partition_name)
		case "odm_dlkm":
			addEntryToPartitionToName(p, s.partitionProps.Odm_dlkm_partition_name)
		default:
			ctx.ModuleErrorf("current partition %s not a super image supported partition", p)
		}
	}

	ctx.VisitDirectDeps(func(m android.Module) {
		if p, ok := nameToPartition[m.Name()]; ok {
			if output, ok := android.OtherModuleProvider(ctx, m, android.OutputFilesProvider); ok {
				partitionToImagePath[p] = output.DefaultOutputFiles[0].String()
			}
		} else if systemOtherPartitionNameNeeded != "" && m.Name() == systemOtherPartitionNameNeeded {
			if output, ok := android.OtherModuleProvider(ctx, m, android.OutputFilesProvider); ok {
				partitionToImagePath["system_other"] = output.DefaultOutputFiles[0].String()
			}
		}
	})

	for _, p := range android.SortedKeys(partitionToImagePath) {
		addStr(p+"_image", partitionToImagePath[p])
	}

	miscInfo := android.PathForModuleOut(ctx, "misc_info.txt")
	android.WriteFileRule(ctx, miscInfo, miscInfoString.String())
	return miscInfo
}
