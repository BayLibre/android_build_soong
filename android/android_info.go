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
	"github.com/google/blueprint/proptools"
)

type androidInfoProperties struct {
	// Name of output file. Defaults to module name
	Stem *string

	// Paths of board-info.txt files.
	Board_info_files []string `android:"path"`

	// Name of bootloader board. If board_info_files is empty, `board={bootloader_board_name}` will
	// be printed to output. Ignored if board_info_files is not empty.
	Bootloader_board_name *string
}

type androidInfoModule struct {
	ModuleBase

	properties androidInfoProperties

	outputFilePath  OutputPath
	installFilePath InstallPath
}

func (p *androidInfoModule) stem() string {
	return proptools.String(p.properties.Stem)
}

func (p *androidInfoModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	outName := proptools.StringDefault(p.properties.Stem, ctx.ModuleName())
	p.outputFilePath = PathForModuleOut(ctx, outName).OutputPath

	rule := NewRuleBuilder(pctx, ctx)

	if boardInfoFiles := PathsForModuleSrc(ctx, p.properties.Board_info_files); len(boardInfoFiles) > 0 {
		txt := PathForModuleGen(ctx, "board-info.txt")
		rule.Command().Text("cat").Inputs(boardInfoFiles).FlagWithOutput("> ", txt)
		rule.Temporary(txt)
		rule.Command().Text("grep").FlagWithArg("-v ", "'#'").Input(txt).FlagWithOutput("> ", p.outputFilePath)
	} else if bootloaderBoardName := proptools.String(p.properties.Bootloader_board_name); bootloaderBoardName != "" {
		rule.Command().Text("echo").Text("'board="+bootloaderBoardName+"'").FlagWithOutput("> ", p.outputFilePath)
	} else {
		rule.Command().Text("echo").Text("''").FlagWithOutput("> ", p.outputFilePath)
	}

	rule.Build(ctx.ModuleName(), "generating android-info.txt")

	ctx.SetOutputFiles(Paths{p.outputFilePath}, "")
}

// android_info module generate a file named android-info.txt that contains various information
// about the device we're building for.  This file is typically packaged up with everything else.
func androidInfoFactory() Module {
	module := &androidInfoModule{}
	module.AddProperties(&module.properties)
	InitAndroidModule(module)
	return module
}
