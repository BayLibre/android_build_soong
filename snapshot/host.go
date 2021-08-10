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

package snapshot

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

//
//  The host_tools module creates a snapshot of the tools defined by the module's deps property.
// This snapshot contains the binaries and any transitive PackagingSpecs, along with a JSON
// meta file.  The snapshot can be installed into another source tree via
// development/vendor_snapshot/update.py, the included modules are provided as preferred
// prebuilts.
//

func init() {
	registerHostBuildComponents(android.InitRegistrationContext)
}

func registerHostBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("host_tools", hostToolsFactory)
}

// Relative installation path
type RelativeInstallPath interface {
	RelativeInstallPath() string
}

type hostTools struct {
	android.ModuleBase
	android.PackagingBase

	zipFile    android.OptionalPath
	installDir android.InstallPath
}

func hostToolsFactory() android.Module {
	module := &hostTools{}
	initHostToolsModule(module)
	return module
}
func initHostToolsModule(module *hostTools) {
	android.InitPackageModule(module)
	android.InitAndroidMultiTargetsArchModule(module, android.HostSupported, android.MultilibCommon)
}

var dependencyTag = struct {
	blueprint.BaseDependencyTag
	android.InstallAlwaysNeededDependencyTag
	android.PackagingItemAlwaysDepTag
}{}

func (f *hostTools) DepsMutator(ctx android.BottomUpMutatorContext) {
	f.AddDeps(ctx, dependencyTag)
}
func (f *hostTools) installFileName() string {
	return f.Name() + ".zip"
}

// Create zipfile with JSON description, notice files... for dependent modules
func (f *hostTools) CreateMetaData(ctx android.ModuleContext, fileName string) android.OutputPath {
	var jsonData []SnapshotJsonFlags
	var metaPaths android.Paths

	metaZipFile := android.PathForModuleOut(ctx, fileName).OutputPath

	// Create JSON file based on the direct dependencies
	ctx.VisitDirectDeps(func(dep android.Module) {
		desc := hostBinJsonDesc(dep)
		if desc != nil {
			jsonData = append(jsonData, *desc)
		}
		if len(dep.EffectiveLicenseFiles()) > 0 {
			noticeFile := android.PathForModuleOut(ctx, "NOTICE_FILES", dep.Name()+".txt").OutputPath
			android.CatFileRule(ctx, dep.EffectiveLicenseFiles(), noticeFile)
			metaPaths = append(metaPaths, noticeFile)
		}

	})
	// Sort notice paths and json data for repeatble build
	sort.Slice(jsonData, func(i, j int) bool {
		return (jsonData[i].ModuleName < jsonData[j].ModuleName)
	})
	sort.Slice(metaPaths, func(i, j int) bool {
		return (metaPaths[i].String() < metaPaths[j].String())
	})

	marsh, err := json.Marshal(jsonData)
	if err != nil {
		ctx.ModuleErrorf("host snapshot json marshal failure: %#v", err)
		return android.OutputPath{}
	}

	jsonZipFile := android.PathForModuleOut(ctx, "host_tools.json").OutputPath
	metaPaths = append(metaPaths, jsonZipFile)
	rspFile := android.PathForModuleOut(ctx, "host_tools.rsp").OutputPath
	android.WriteFileRule(ctx, jsonZipFile, string(marsh))

	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().Text("mkdir").Flag("-p").Text(android.PathForModuleOut(ctx).String())

	builder.Command().
		BuiltTool("soong_zip").
		FlagWithArg("-C ", android.PathForModuleOut(ctx).OutputPath.String()).
		FlagWithOutput("-o ", metaZipFile).
		FlagWithRspFileInputList("-r ", rspFile, metaPaths)
	builder.Build("zip_meta", fmt.Sprintf("zipping meta data for %s", ctx.ModuleName()))

	return metaZipFile
}

// Create the host tool zip file
func (f *hostTools) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Create a zip file for the binaries, and a zip of the meta data, then merge zips
	depsZipFile := android.PathForModuleOut(ctx, f.Name()+"_deps.zip").OutputPath
	modsZipFile := android.PathForModuleOut(ctx, f.Name()+"_mods.zip").OutputPath
	outputFile := android.PathForModuleOut(ctx, f.installFileName()).OutputPath

	f.installDir = android.PathForModuleInstall(ctx)

	f.CopyDepsToZip(ctx, depsZipFile)

	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().Text("mkdir").Flag("-p").Text(android.PathForModuleOut(ctx).String())
	builder.Command().
		BuiltTool("zip2zip").
		FlagWithInput("-i ", depsZipFile).
		FlagWithOutput("-o ", modsZipFile).
		Text("**/*:" + proptools.ShellEscape(f.installDir.String()))

	metaZipFile := f.CreateMetaData(ctx, f.Name()+"_meta.zip")

	builder.Command().
		BuiltTool("merge_zips").
		Output(outputFile).
		Input(metaZipFile).
		Input(modsZipFile)

	builder.Build("manifest", fmt.Sprintf("Adding manifest %s", f.installFileName()))
	zip := ctx.InstallFile(f.installDir, f.installFileName(), outputFile)
	f.zipFile = android.OptionalPathForPath(zip)

}

// Implements android.AndroidMkEntriesProvider
func (f *hostTools) AndroidMkEntries() []android.AndroidMkEntries {
	if !f.zipFile.Valid() {
		return []android.AndroidMkEntries{}
	}

	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: f.zipFile,
		DistFiles:  android.MakeDefaultDistFiles(f.zipFile.Path()),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", f.installDir.ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", f.installFileName())
			},
		},
	}}
}

// Get host tools path and relative install string helpers
func hostBinToolPath(m android.Module) android.OptionalPath {
	switch t := m.(type) {
	case android.HostToolProvider:
		return t.HostToolPath()
		break
	}
	return android.OptionalPath{}

}
func hostRelativePathString(m android.Module) string {
	var outString string
	if rel, ok := m.(RelativeInstallPath); ok {
		outString = rel.RelativeInstallPath()
	}
	return outString
}

// Create JSON description for given module, only create descriptions for binary modueles which
// provide a valid HostToolPath
func hostBinJsonDesc(m android.Module) *SnapshotJsonFlags {
	path := hostBinToolPath(m)
	relPath := hostRelativePathString(m)
	if path.Valid() && path.String() != "" {
		return &SnapshotJsonFlags{
			ModuleName:          m.Name(),
			ModuleStemName:      filepath.Base(path.String()),
			Filename:            path.String(),
			Required:            append(m.HostRequiredModuleNames(), m.RequiredModuleNames()...),
			RelativeInstallPath: relPath,
		}
	}
	return nil
}
