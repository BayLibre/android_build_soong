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
	"android/soong/android"
	"strconv"

	"github.com/google/blueprint/proptools"
)

type soongGeneratedPartitionProperties struct {
	// Name of the filesystem module to be generated for this partition
	Name *string

	// Whether the generation of this partition is enabled or not
	Enabled *bool
}

type filesystemCreatorProperties struct {
	// The properties specific to the system partition filesystem module to be generated
	System soongGeneratedPartitionProperties
}

type filesystemCreator struct {
	android.ModuleBase

	filesystemCreatorProperties filesystemCreatorProperties
}

func filesystemCreatorFactory() android.Module {
	module := &filesystemCreator{}

	module.AddProperties(&module.filesystemCreatorProperties)
	android.AddLoadHook(module, func(ctx android.LoadHookContext) {
		module.createInternalModules(ctx)
	})

	return module
}

func (f *filesystemCreator) createInternalModules(ctx android.LoadHookContext) {
	if proptools.BoolDefault(f.filesystemCreatorProperties.System.Enabled, false) {
		f.createSystemImage(ctx)
	}
}

func (f *filesystemCreator) createSystemImage(ctx android.LoadHookContext) {
	props := &filesystemProperties{}
	partitionVars := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse
	systemPartitionVars := partitionVars.PartitionQualifiedVariables["system"]

	//  BOARD_AVB_ENABLE
	props.Use_avb = proptools.BoolPtr(partitionVars.BoardAvbEnable)
	// BOARD_AVB_KEY_PATH
	props.Avb_private_key = proptools.StringPtr(systemPartitionVars.BoardAvbKeyPath)
	// BOARD_AVB_ALGORITHM
	props.Avb_algorithm = proptools.StringPtr(systemPartitionVars.BoardAvbAlgorithm)
	// BOARD_AVB_SYSTEM_ROLLBACK_INDEX
	if rollbackIndex, err := strconv.ParseInt(systemPartitionVars.BoardAvbRollbackIndex, 10, 64); err == nil {
		props.Rollback_index = proptools.Int64Ptr(rollbackIndex)
	}

	props.Partition_name = proptools.StringPtr("system")
	// BOARD_SYSTEMIMAGE_FILE_SYSTEM_TYPE
	props.Type = proptools.StringPtr(systemPartitionVars.BoardFileSystemType)
	// Must match one of [validPartitions]
	props.Partition_type = proptools.StringPtr("system")

	props.Base_dir = proptools.StringPtr("system")

	props.Gen_aconfig_flags_pb = proptools.BoolPtr(true)

	ctx.CreateModule(systemImageFactory, props)
}

func (f *filesystemCreator) GenerateAndroidBuildActions(ctx android.ModuleContext) {

}
