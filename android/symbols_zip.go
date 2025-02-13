// Copyright (C) 2025 The Android Open Source Project
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

func init() {
	pctx.HostBinToolVariable("merge_zips", "merge_zips")
	pctx.HostBinToolVariable("symbols_map", "symbols_map")

	InitRegistrationContext.RegisterSingletonModuleType("symbols_zip", SymbolsZipSingletonModuleFactory)
}

type AndroidDeviceInfo struct {
	SymbolsZip     Path
	SymbolsMapping Path
	MainDevice     bool
}

var AndroidDeviceInfoProvider = blueprint.NewProvider[AndroidDeviceInfo]()

type SymbolsZipSingletonModuleProperties struct {
	Test_suites []string
}

type SymbolsZipSingletonModule struct {
	SingletonModuleBase

	properties SymbolsZipSingletonModuleProperties

	symbolsZipFile     ModuleOutPath
	symbolsMappingFile ModuleOutPath
}

func SymbolsZipSingletonModuleFactory() SingletonModule {
	module := &SymbolsZipSingletonModule{}
	module.AddProperties(&module.properties)
	InitAndroidArchModule(module, DeviceSupported, MultilibCommon)
	return module
}

func (sz *SymbolsZipSingletonModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	sz.symbolsZipFile = PathForModuleOut(ctx, "symbols.zip")
	sz.symbolsMappingFile = PathForModuleOut(ctx, "symbols-mapping.textproto")

	namePrefix := ""
	if ctx.Config().HasDeviceProduct() {
		namePrefix = ctx.Config().DeviceProduct() + "-"
	}

	if !ctx.Config().KatiEnabled() {
		ctx.DistForGoalWithFilename("droidcore-unbundled", sz.symbolsZipFile, fmt.Sprintf("%ssymbols-FILE_NAME_TAG_PLACEHOLDER.zip", namePrefix))
		ctx.DistForGoalWithFilename("droidcore-unbundled", sz.symbolsMappingFile, fmt.Sprintf("%ssymbols-mapping-FILE_NAME_TAG_PLACEHOLDER.textproto", namePrefix))
	}
}

func (sz *SymbolsZipSingletonModule) GenerateSingletonBuildActions(ctx SingletonContext) {
	var deviceSymbolsZip, deviceSymbolsMapping Path
	ctx.VisitAllModuleProxies(func(proxy ModuleProxy) {
		if adi, ok := OtherModuleProvider(ctx, proxy, AndroidDeviceInfoProvider); ok && adi.MainDevice {
			deviceSymbolsZip = adi.SymbolsZip
			deviceSymbolsMapping = adi.SymbolsMapping
		}
	})

	// TODO(jihoonkang): Merge symbols and mapping of test suites modules once they are
	// available in Soong.
	rule := NewRuleBuilder(pctx, ctx)

	symbolsZipCmd := rule.Command()
	symbolsZipCmd.BuiltTool("merge_zips").Output(sz.symbolsZipFile).Input(deviceSymbolsZip)

	symbolsMappingCmd := rule.Command()
	symbolsMappingCmd.BuiltTool("symbols_map").Flag("-merge").Output(sz.symbolsMappingFile).Input(deviceSymbolsMapping)

	rule.Build("symbols_zip_build", "build symbols.zip and symbols-mapping.textproto")
}
