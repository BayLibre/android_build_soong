// Copyright 2021 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package etc

import (
	"android/soong/android"
	"fmt"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func RegisterSnapshotEtcModule(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("snapshot_etc", SnapshotEtcFactory)
}

func init() {
	RegisterSnapshotEtcModule(android.InitRegistrationContext)
}

// prebuilt_etc is for a prebuilt artifact that is installed in
// <partition>/etc/<sub_dir> directory.
func SnapshotEtcFactory() android.Module {
	module := &SnapshotEtc{}
	module.AddProperties(&module.properties)
	// module.prebuilt.ForcePrefer()

	var srcsSupplier = func(_ android.BaseModuleContext, prebuilt android.Module) []string {
		s, ok := prebuilt.(*SnapshotEtc)
		if !ok || s.properties.Src == nil {
			return []string{}
		}

		return []string{*s.properties.Src}
	}

	// This module is device-only
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibFirst)
	android.InitPrebuiltModuleWithSrcSupplier(module, srcsSupplier, "src")
	return module
}

type snapshotEtcProperties struct {
	Src                   *string `android:"path,arch_variant"`
	Filename              *string `android:"arch_variant"`
	Relative_install_path *string `android:"arch_variant"`
}

type SnapshotEtc struct {
	android.ModuleBase
	prebuilt   android.Prebuilt
	properties snapshotEtcProperties

	outputFilePath android.OutputPath
	installDirPath android.InstallPath
}

func (s *SnapshotEtc) Prebuilt() *android.Prebuilt {
	return &s.prebuilt
}

func (s *SnapshotEtc) Name() string {
	return s.prebuilt.Name(s.BaseModuleName())
}

func (s *SnapshotEtc) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if s.properties.Src == nil {
		ctx.PropertyErrorf("src", "missing prebuilt source file")
		return
	}

	fmt.Println("CLOWCARD: GenerateAndroidBuildActions called!")

	sourceFilePath := android.PathForModuleSrc(ctx, proptools.String(s.properties.Src))

	// Determine the output file basename.
	// If Filename is set, use the name specified by the property.
	// If Filename_from_src is set, use the source file name.
	// Otherwise use the module name.
	filename := proptools.String(s.properties.Filename)
	if filename == "" {
		filename = ctx.ModuleName()
	}

	s.outputFilePath = android.PathForModuleOut(ctx, filename).OutputPath

	if strings.Contains(filename, "/") {
		ctx.PropertyErrorf("filename", "filename cannot contain separator '/'")
		return
	}

	// If soc install dir was specified and SOC specific is set, set the installDirPath to the
	// specified socInstallDirBase.
	installBaseDir := "etc"
	s.installDirPath = android.PathForModuleInstall(ctx, installBaseDir)

	// This ensures that outputFilePath has the correct name for others to
	// use, as the source file may have a different name.
	ctx.Build(pctx, android.BuildParams{
		Rule:   android.Cp,
		Output: s.outputFilePath,
		Input:  sourceFilePath,
	})

	// Call InstallFile even when uninstallable to make the module included in the package
	ctx.InstallFile(s.installDirPath, s.outputFilePath.Base(), sourceFilePath)
}

func (p *SnapshotEtc) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class: "ETC",
		// OverrideName: p.BaseModuleName(),
		OutputFile: android.OptionalPathForPath(p.outputFilePath),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_TAGS", "optional")
				entries.SetString("LOCAL_MODULE_PATH", p.installDirPath.ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", p.outputFilePath.Base())
			},
		},
	}}
}

type snapshotEtcDependencyTag struct {
	blueprint.DependencyTag
}

var tag = snapshotEtcDependencyTag{}

// func (t *snapshotEtcDependencyTag) ReplaceSourceWithPrebuilt() bool {
// 	return true
// }

// Add other dependencies as normal.
// func (s *SnapshotEtc) DepsMutator(ctx android.BottomUpMutatorContext) {

// 	moduleName := s.BaseModuleName()
// 	if ctx.OtherModuleExists(moduleName) {
// 		ctx.AddVariationDependencies(nil, tag, moduleName)
// 	}
// }

func (s *SnapshotEtc) CoreVariantNeeded(ctx android.BaseModuleContext) bool {
	return !s.ModuleBase.InstallInRecovery() && !s.ModuleBase.InstallInRamdisk() &&
		!s.ModuleBase.InstallInVendorRamdisk() && !s.ModuleBase.InstallInDebugRamdisk()
}

func (p *SnapshotEtc) RamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return p.ModuleBase.InstallInRamdisk()
}

func (p *SnapshotEtc) VendorRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return p.ModuleBase.InstallInVendorRamdisk()
}

func (p *SnapshotEtc) DebugRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return p.ModuleBase.InstallInDebugRamdisk()
}

func (p *SnapshotEtc) RecoveryVariantNeeded(ctx android.BaseModuleContext) bool {
	return p.ModuleBase.InstallInRecovery()
}

func (p *SnapshotEtc) ExtraImageVariations(ctx android.BaseModuleContext) []string {
	return nil
}

func (p *SnapshotEtc) SetImageVariation(ctx android.BaseModuleContext, variation string, module android.Module) {
}

func (p *SnapshotEtc) ImageMutatorBegin(ctx android.BaseModuleContext) {}

func (p *SnapshotEtc) OutputFiles(tag string) (android.Paths, error) {
	return android.Paths{p.outputFilePath}, nil
}

var _ android.PrebuiltInterface = (*SnapshotEtc)(nil)
var _ android.ImageInterface = (*SnapshotEtc)(nil)
