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

package fsgen

import (
	"android/soong/android"
	"android/soong/filesystem"
	"github.com/google/blueprint/proptools"
)

func buildingSuperImage(partitionVars android.PartitionVariables) bool {
	return partitionVars.ProductBuildSuperPartition
}

func createSuperImage(ctx android.LoadHookContext, partitions []string, partitionVars android.PartitionVariables) {
	baseProps := &struct {
		Name *string
	}{
		Name: proptools.StringPtr(generatedModuleName(ctx.Config(), "super_image")),
	}

	superImageProps := &filesystem.SuperImageProperties{
		Size:                     proptools.StringPtr(partitionVars.BoardSuperPartitionSize),
		MetadataDevice:           proptools.StringPtr(partitionVars.BoardSuperPartitionMetadataDevice),
		BlockDevices:             proptools.StringPtr(partitionVars.BoardSuperPartitionBlockDevices[0]),
		AbUpdate:                 proptools.BoolPtr(partitionVars.AbOtaUpdater),
		Retrofit:                 proptools.BoolPtr(partitionVars.ProductRetrofitDynamicPartitions),
		VirtualAb:                proptools.BoolPtr(partitionVars.ProductVirtualAbOta),
		VirtualAbRetrofit:        proptools.BoolPtr(partitionVars.ProductVirtualAbOtaRetrofit),
		BuildingSystemOtherImage: proptools.BoolPtr(partitionVars.BuildingSystemOtherImage),
		PartitionList:            partitionVars.BoardSuperPartitionPartitionList,
		PartitionGroups:          partitionVars.BoardSuperPartitionGroups,
		PartitionGroupsInfo:      partitionVars.BoardSuperPartitionGroupsInfo,
		UseDynamicPartitions:     proptools.BoolPtr(partitionVars.ProductUseDynamicPartitions),
		LpMakeDir:                proptools.StringPtr(partitionVars.LpMakeDir),
	}
	sparse := !partitionVars.TargetUserimagesSparseExtDisabled && !partitionVars.TargetUserimagesSparseF2fsDisabled
	superImageProps.Sparse = proptools.BoolPtr(sparse)

	partitionNameProps := &filesystem.SuperImagePartitionNameProperties{}
	if android.InList("system", partitions) {
		partitionNameProps.System_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "system"))
	}
	if android.InList("system_ext", partitions) {
		partitionNameProps.System_ext_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "system_ext"))
	}
	if android.InList("system_dlkm", partitions) {
		partitionNameProps.System_dlkm_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "system_dlkm"))
	}
	if android.InList("system_other", partitions) {
		partitionNameProps.System_other_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "system_other"))
	}
	if android.InList("product", partitions) {
		partitionNameProps.Product_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "product"))
	}
	if android.InList("vendor", partitions) {
		partitionNameProps.Vendor_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "vendor"))
	}
	if android.InList("vendor_dlkm", partitions) {
		partitionNameProps.Vendor_dlkm_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "vendor_dlkm"))
	}
	if android.InList("odm", partitions) {
		partitionNameProps.Odm_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "odm"))
	}
	if android.InList("odm_dlkm", partitions) {
		partitionNameProps.Odm_dlkm_partition_name = proptools.StringPtr(generatedModuleNameForPartition(ctx.Config(), "odm_dlkm"))
	}

	ctx.CreateModule(filesystem.SuperImageFactory, baseProps, superImageProps, partitionNameProps)
}
