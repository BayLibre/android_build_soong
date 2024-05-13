// Copyright 2024 The Android Open Source Project
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
)

func init() {
	ctx := InitRegistrationContext
	ctx.RegisterParallelSingletonModuleType("android_info", androidInfoFactory)
}

type androidInfoModule struct {
	SingletonModuleBase

	outputFilePath  OutputPath
	installFilePath InstallPath
}

var _ OutputFileProducer = (*androidInfoModule)(nil)

// OutputFileProducer
func (p *androidInfoModule) OutputFiles(tag string) (Paths, error) {
	if tag != "" {
		return nil, fmt.Errorf("unsupported tag %q", tag)
	}
	return Paths{p.outputFilePath}, nil
}

func (p *androidInfoModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	p.outputFilePath = PathForModuleOut(ctx, ctx.ModuleName()).OutputPath

	rule := NewRuleBuilder(pctx, ctx)

	config := ctx.Config()
	var boardInfoTxt OptionalPath
	if boardInfoFiles := config.BoardInfoFiles(ctx); len(boardInfoFiles) > 0 {
		txt := PathForModuleGen(ctx, "board-info.txt")
		rule.Command().Text("cat").Inputs(boardInfoFiles).FlagWithOutput("> ", txt)
		boardInfoTxt = OptionalPathForPath(txt)
	} else if boardInfoFile := config.BoardInfoFile(ctx); boardInfoFile.Valid() {
		boardInfoTxt = boardInfoFile
	} else {
		boardInfoTxt = ExistentPathForSource(ctx, config.DeviceDir(ctx).String(), "board-info.txt")
	}

	// TODO: add check_radio_versions check here.

	if boardInfoTxt.Valid() {
		rule.Command().Text("grep").FlagWithArg("-v ", "'#'").Input(boardInfoTxt.Path()).FlagWithOutput("> ", p.outputFilePath)
	} else if bootloaderBoardName := config.BootloaderBoardName(); bootloaderBoardName != "" {
		rule.Command().Text("echo").Text("'board="+bootloaderBoardName+"'").FlagWithOutput("> ", p.outputFilePath)
	} else {
		rule.Command().Text("echo").Text("''").FlagWithOutput("> ", p.outputFilePath)
	}

	rule.Build(ctx.ModuleName(), "generating android-info.txt")

	// TODO: install android-info.txt to $(PRODUCT_OUT).
}

func (p *androidInfoModule) GenerateSingletonBuildActions(ctx SingletonContext) {
	// does nothing
}

// android_info module generate a file named android-info.txt that contains various information
// about the device we're building for.  This file is typically packaged up with everything else.
//
// The following logic is used to find the contents of the info file:
//  1. TARGET_BOARD_INFO_FILES (can be set in BoardConfig.mk) will be combined.
//  2. TARGET_BOARD_INFO_FILE (can be set in BoardConfig.mk) will be used.
//  3. $(TARGET_DEVICE_DIR)/board-info.txt will be used if present.
//
// Specifying both TARGET_BOARD_INFO_FILES and TARGET_BOARD_INFO_FILE is an error.
func androidInfoFactory() SingletonModule {
	module := &androidInfoModule{}
	InitAndroidModule(module)
	return module
}
