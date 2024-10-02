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
	android.InitAndroidMultiTargetsArchModule(module, android.DeviceSupported, android.MultilibCommon)
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
	baseProps := &struct {
		Name *string
	}{
		Name: f.filesystemCreatorProperties.System.Name,
	}

	fsProps := &filesystemProperties{}
	partitionVars := ctx.Config().ProductVariables().PartitionVarsForSoongMigrationOnlyDoNotUse
	systemPartitionVars := partitionVars.PartitionQualifiedVariables["system"]

	//  BOARD_AVB_ENABLE
	fsProps.Use_avb = proptools.BoolPtr(partitionVars.BoardAvbEnable)
	// BOARD_AVB_KEY_PATH
	fsProps.Avb_private_key = proptools.StringPtr(systemPartitionVars.BoardAvbKeyPath)
	// BOARD_AVB_ALGORITHM
	fsProps.Avb_algorithm = proptools.StringPtr(systemPartitionVars.BoardAvbAlgorithm)
	// BOARD_AVB_SYSTEM_ROLLBACK_INDEX
	if rollbackIndex, err := strconv.ParseInt(systemPartitionVars.BoardAvbRollbackIndex, 10, 64); err == nil {
		fsProps.Rollback_index = proptools.Int64Ptr(rollbackIndex)
	}

	fsProps.Partition_name = proptools.StringPtr("system")
	// BOARD_SYSTEMIMAGE_FILE_SYSTEM_TYPE
	fsProps.Type = proptools.StringPtr(systemPartitionVars.BoardFileSystemType)

	fsProps.Base_dir = proptools.StringPtr("system")

	fsProps.Gen_aconfig_flags_pb = proptools.BoolPtr(true)

	// Identical to that of the generic_system_image
	fsProps.Fsverity.Inputs = []string{
		"etc/boot-image.prof",
		"etc/dirty-image-objects",
		"etc/preloaded-classes",
		"etc/classpaths/*.pb",
		"framework/*",
		"framework/*/*",     // framework/{arch}
		"framework/oat/*/*", // framework/oat/{arch}
	}

	// system_image properties that are not set:
	// - filesystemProperties.Avb_hash_algorithm
	// - filesystemProperties.File_contexts
	// - filesystemProperties.Dirs
	// - filesystemProperties.Symlinks
	// - filesystemProperties.Fake_timestamp
	// - filesystemProperties.Uuid
	// - filesystemProperties.Mount_point
	// - filesystemProperties.Include_make_built_files
	// - filesystemProperties.Build_logtags
	// - filesystemProperties.Fsverity.Libs
	// - systemImageProperties.Linker_config_src
	ctx.CreateModule(systemImageFactory, baseProps, fsProps)
}

func (f *filesystemCreator) GenerateAndroidBuildActions(ctx android.ModuleContext) {

}
